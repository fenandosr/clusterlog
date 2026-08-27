package clusterlog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const siteStateVersion = 2

func loadReviewLog(root string) (ReviewLog, error) {
	log := ReviewLog{Version: 1, Events: []ReviewEvent{}}
	files, err := reviewEventFiles(root)
	if err != nil {
		return log, err
	}
	for _, path := range files {
		var event ReviewEvent
		if err := loadJSON(path, &event); err != nil {
			return log, err
		}
		log.Events = append(log.Events, event)
	}
	sortReviewEvents(log.Events)
	return log, nil
}

func loadTaskLog(root string) (TaskLog, error) {
	log := TaskLog{Version: 1, Events: []TaskEvent{}}
	files, err := taskEventFiles(root)
	if err != nil {
		return log, err
	}
	for _, path := range files {
		var event TaskEvent
		if err := loadJSON(path, &event); err != nil {
			return log, err
		}
		log.Events = append(log.Events, event)
	}
	sortTaskEvents(log.Events)
	return log, nil
}

func requiredReviewers(registry AdminRegistry, policy, authorID string) []string {
	policy = firstNonEmpty(policy, registry.ReviewPolicy.Default)
	required := activeAdminIDs(registry)
	excludeAuthor := policy == "all-active-except-author" ||
		(policy == "all-active" && !registry.ReviewPolicy.IncludeAuthor)
	if !excludeAuthor || authorID == "" {
		return required
	}
	filtered := make([]string, 0, len(required))
	for _, id := range required {
		if id != authorID {
			filtered = append(filtered, id)
		}
	}
	return filtered
}

func (a *App) BuildState(root string) (SiteState, error) {
	registry, err := loadAdmins(root)
	if err != nil {
		return SiteState{}, err
	}
	reviews, err := loadReviewLog(root)
	if err != nil {
		return SiteState{}, err
	}
	tasksLog, err := loadTaskLog(root)
	if err != nil {
		return SiteState{}, err
	}

	memoryFiles, err := listMarkdownFiles(root, "memorias")
	if err != nil {
		return SiteState{}, err
	}
	taskFiles, err := listMarkdownFiles(root, "tareas")
	if err != nil {
		return SiteState{}, err
	}
	projectFiles, err := listMarkdownFiles(root, "proyectos")
	if err != nil {
		return SiteState{}, err
	}

	reviewFiles, err := reviewEventFiles(root)
	if err != nil {
		return SiteState{}, err
	}
	taskEventPaths, err := taskEventFiles(root)
	if err != nil {
		return SiteState{}, err
	}
	sourcePaths := []string{filepath.Join(root, "data", "admins.json")}
	sourcePaths = append(sourcePaths, reviewFiles...)
	sourcePaths = append(sourcePaths, taskEventPaths...)
	sourcePaths = append(sourcePaths, memoryFiles...)
	sourcePaths = append(sourcePaths, taskFiles...)
	sourcePaths = append(sourcePaths, projectFiles...)
	digest, err := sourceDigest(sourcePaths)
	if err != nil {
		return SiteState{}, err
	}

	memories := make([]MemoryRecord, 0, len(memoryFiles))
	for _, path := range memoryFiles {
		meta, err := parseMarkdown(path)
		if err != nil {
			return SiteState{}, err
		}
		id := normalizeID(meta.ExtraString("id"))
		if id == "" {
			continue
		}
		rel, _ := filepath.Rel(root, path)
		record := MemoryRecord{
			EntryID:           id,
			Title:             meta.String("title"),
			Description:       meta.String("description"),
			Date:              meta.String("date"),
			AuthorID:          meta.ExtraString("author_id"),
			AuthorFingerprint: meta.ExtraString("author_fingerprint"),
			Risk:              firstNonEmpty(meta.ExtraString("risk"), "low"),
			Ticket:            meta.ExtraString("ticket"),
			ReviewRequired:    meta.ExtraBool("review_required"),
			ReviewPolicy:      meta.ExtraString("review_policy"),
			ReviewerSnapshot:  meta.ExtraBool("reviewer_snapshot"),
			Reviewers:         uniqueSorted(meta.ExtraStrings("reviewers")),
			Tags:              uniqueSorted(meta.Taxonomies["tags"]),
			Systems:           uniqueSorted(meta.Taxonomies["systems"]),
			Slug:              pageSlug(meta),
			SourcePath:        filepath.ToSlash(rel),
			ContentSHA256:     meta.ContentHash,
		}
		record.Permalink = "/memorias/" + record.Slug + "/"
		if record.ReviewPolicy == "" {
			record.ReviewPolicy = registry.ReviewPolicy.Default
		}
		memories = append(memories, record)
	}
	sort.Slice(memories, func(i, j int) bool {
		if memories[i].Date == memories[j].Date {
			return memories[i].EntryID > memories[j].EntryID
		}
		return memories[i].Date > memories[j].Date
	})

	activeIDs := activeAdminIDs(registry)
	queue := []ReviewQueueItem{}
	for _, memory := range memories {
		if !memory.ReviewRequired {
			continue
		}
		var required []string
		if memory.ReviewerSnapshot {
			required = append([]string(nil), memory.Reviewers...)
		} else {
			required = requiredReviewers(registry, memory.ReviewPolicy, memory.AuthorID)
		}

		reviewedSet := map[string]bool{}
		for _, event := range reviews.Events {
			if normalizeID(event.EntryID) != memory.EntryID || event.ContentSHA256 != memory.ContentSHA256 {
				continue
			}
			if contains(required, event.AdminID) {
				reviewedSet[event.AdminID] = true
			}
		}
		reviewed := []string{}
		missing := []string{}
		for _, id := range required {
			if reviewedSet[id] {
				reviewed = append(reviewed, id)
			} else {
				missing = append(missing, id)
			}
		}
		item := ReviewQueueItem{
			EntryID: memory.EntryID, Title: memory.Title, Description: memory.Description,
			Date: memory.Date, AuthorID: memory.AuthorID, Risk: memory.Risk,
			Tags: memory.Tags, Systems: memory.Systems, Permalink: memory.Permalink,
			ContentSHA256: memory.ContentSHA256, ReviewedBy: reviewed,
			MissingReviewers: missing, RequiredCount: len(required), ReviewedCount: len(reviewed),
		}
		if !memory.ReviewerSnapshot && len(activeIDs) == 0 {
			item.ConfigurationError = "No hay administradores activos; registre al menos uno antes de cerrar revisiones."
		}
		if item.ConfigurationError != "" || len(missing) > 0 {
			queue = append(queue, item)
		}
	}

	taskStates := make([]TaskState, 0, len(taskFiles))
	taskIndex := map[string]int{}
	for _, path := range taskFiles {
		meta, err := parseMarkdown(path)
		if err != nil {
			return SiteState{}, err
		}
		id := normalizeID(meta.ExtraString("id"))
		if id == "" {
			continue
		}
		rel, _ := filepath.Rel(root, path)
		state := TaskState{
			TaskID:          id,
			RunningBy:       []string{},
			Title:           meta.String("title"),
			Description:     meta.String("description"),
			Status:          firstNonEmpty(meta.ExtraString("status"), "todo"),
			Priority:        firstNonEmpty(meta.ExtraString("priority"), "medium"),
			ProjectID:       normalizeID(meta.ExtraString("project_id")),
			Assignees:       uniqueSorted(meta.ExtraStrings("assignees")),
			EstimateMinutes: meta.ExtraInt("estimate_minutes"),
			Due:             meta.ExtraString("due"),
			Slug:            pageSlug(meta),
			SourcePath:      filepath.ToSlash(rel),
		}
		state.Permalink = "/tareas/" + state.Slug + "/"
		taskIndex[id] = len(taskStates)
		taskStates = append(taskStates, state)
	}
	running := map[string]map[string]bool{}
	for _, event := range tasksLog.Events {
		id := normalizeID(event.TaskID)
		idx, ok := taskIndex[id]
		if !ok {
			continue
		}
		state := &taskStates[idx]
		actor := event.AdminID
		switch event.Type {
		case "start":
			if running[id] == nil {
				running[id] = map[string]bool{}
			}
			running[id][actor] = true
			if state.Status == "todo" || state.Status == "backlog" {
				state.Status = "in_progress"
			}
		case "stop":
			state.ActualMinutes += event.Minutes
			if running[id] != nil {
				delete(running[id], actor)
			}
		case "adjust":
			state.ActualMinutes += event.Minutes
		case "status":
			if event.Status != "" {
				state.Status = event.Status
			}
		}
	}
	for id, actors := range running {
		idx, ok := taskIndex[id]
		if !ok {
			continue
		}
		for actor, isRunning := range actors {
			if isRunning {
				taskStates[idx].RunningBy = append(taskStates[idx].RunningBy, actor)
			}
		}
		sort.Strings(taskStates[idx].RunningBy)
	}
	sort.Slice(taskStates, func(i, j int) bool {
		order := map[string]int{"in_progress": 0, "blocked": 1, "todo": 2, "backlog": 3, "done": 4, "cancelled": 5}
		oi, ok := order[taskStates[i].Status]
		if !ok {
			oi = 9
		}
		oj, ok := order[taskStates[j].Status]
		if !ok {
			oj = 9
		}
		if oi == oj {
			if taskStates[i].Due == taskStates[j].Due {
				return taskStates[i].TaskID < taskStates[j].TaskID
			}
			if taskStates[i].Due == "" {
				return false
			}
			if taskStates[j].Due == "" {
				return true
			}
			return taskStates[i].Due < taskStates[j].Due
		}
		return oi < oj
	})

	projects := make([]ProjectState, 0, len(projectFiles))
	for _, path := range projectFiles {
		meta, err := parseMarkdown(path)
		if err != nil {
			return SiteState{}, err
		}
		id := normalizeID(meta.ExtraString("id"))
		if id == "" {
			continue
		}
		rel, _ := filepath.Rel(root, path)
		project := ProjectState{
			ProjectID:   id,
			Title:       meta.String("title"),
			Description: meta.String("description"),
			Status:      firstNonEmpty(meta.ExtraString("status"), "planned"),
			Owners:      uniqueSorted(meta.ExtraStrings("owners")),
			TargetDate:  meta.ExtraString("target_date"),
			Slug:        pageSlug(meta),
			SourcePath:  filepath.ToSlash(rel),
		}
		project.Permalink = "/proyectos/" + project.Slug + "/"
		projects = append(projects, project)
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].ProjectID < projects[j].ProjectID
	})

	openTasks := 0
	for _, task := range taskStates {
		if task.Status != "done" && task.Status != "cancelled" {
			openTasks++
		}
	}
	activeProjects := 0
	for _, project := range projects {
		switch project.Status {
		case "done", "completed", "cancelled", "archived":
		default:
			activeProjects++
		}
	}
	recent := memories
	if len(recent) > 30 {
		recent = recent[:30]
	}
	return SiteState{
		Version:      siteStateVersion,
		SourceSHA256: digest,
		Stats: SiteStats{
			ActiveAdmins: len(activeIDs), PendingReviews: len(queue),
			OpenTasks: openTasks, ActiveProjects: activeProjects,
		},
		ReviewQueue:    queue,
		RecentMemories: recent,
		Tasks:          taskStates,
		Projects:       projects,
	}, nil
}

func (a *App) Sync(root string, check bool) (SiteState, bool, string, error) {
	state, err := a.BuildState(root)
	if err != nil {
		return SiteState{}, false, "", err
	}
	path := filepath.Join(root, "data", "generated", "site_state.json")
	var old SiteState
	oldErr := loadJSON(path, &old)
	upToDate := oldErr == nil && old.Version == state.Version && old.SourceSHA256 == state.SourceSHA256
	if check {
		if !upToDate {
			return state, false, path, NewError(ExitValidation, "generated_state_stale",
				"data/generated/site_state.json no corresponde a las fuentes actuales", map[string]any{
					"expected_source_sha256": state.SourceSHA256,
					"actual_source_sha256":   old.SourceSHA256,
					"hint":                   "ejecute clusterlog sync",
				})
		}
		return old, false, path, nil
	}
	if upToDate {
		return old, false, path, nil
	}
	state.GeneratedAt = a.Options.Now().UTC().Format("2006-01-02T15:04:05Z")
	if a.Options.DryRun {
		return state, true, path, nil
	}
	if err := writeJSONAtomic(path, state); err != nil {
		return SiteState{}, false, path, err
	}
	return state, true, path, nil
}

func findMemory(root, id string) (MemoryRecord, PageMeta, error) {
	id = normalizeID(id)
	files, err := listMarkdownFiles(root, "memorias")
	if err != nil {
		return MemoryRecord{}, PageMeta{}, err
	}
	for _, path := range files {
		meta, err := parseMarkdown(path)
		if err != nil {
			return MemoryRecord{}, PageMeta{}, err
		}
		if normalizeID(meta.ExtraString("id")) == id {
			rel, _ := filepath.Rel(root, path)
			record := MemoryRecord{
				EntryID: id,
				Title:   meta.String("title"), Description: meta.String("description"),
				Date: meta.String("date"), AuthorID: meta.ExtraString("author_id"),
				AuthorFingerprint: meta.ExtraString("author_fingerprint"), Risk: meta.ExtraString("risk"),
				ReviewRequired: meta.ExtraBool("review_required"), ReviewPolicy: meta.ExtraString("review_policy"),
				ReviewerSnapshot: meta.ExtraBool("reviewer_snapshot"), Reviewers: uniqueSorted(meta.ExtraStrings("reviewers")),
				Tags: meta.Taxonomies["tags"], Systems: meta.Taxonomies["systems"],
				Slug: pageSlug(meta), SourcePath: filepath.ToSlash(rel), ContentSHA256: meta.ContentHash,
			}
			record.Permalink = "/memorias/" + record.Slug + "/"
			return record, meta, nil
		}
	}
	return MemoryRecord{}, PageMeta{}, NewError(ExitUsage, "memory_not_found", "no existe la memoria indicada", id)
}

func findTask(root, id string) (TaskState, PageMeta, error) {
	id = normalizeID(id)
	files, err := listMarkdownFiles(root, "tareas")
	if err != nil {
		return TaskState{}, PageMeta{}, err
	}
	for _, path := range files {
		meta, err := parseMarkdown(path)
		if err != nil {
			return TaskState{}, PageMeta{}, err
		}
		if normalizeID(meta.ExtraString("id")) == id {
			rel, _ := filepath.Rel(root, path)
			state := TaskState{
				TaskID: id, Title: meta.String("title"), Description: meta.String("description"),
				Status: firstNonEmpty(meta.ExtraString("status"), "todo"), Priority: meta.ExtraString("priority"),
				ProjectID: normalizeID(meta.ExtraString("project_id")), Assignees: meta.ExtraStrings("assignees"),
				EstimateMinutes: meta.ExtraInt("estimate_minutes"), Due: meta.ExtraString("due"),
				Slug: pageSlug(meta), SourcePath: filepath.ToSlash(rel),
			}
			state.Permalink = "/tareas/" + state.Slug + "/"
			return state, meta, nil
		}
	}
	return TaskState{}, PageMeta{}, NewError(ExitUsage, "task_not_found", "no existe la tarea indicada", id)
}

func taskActiveStart(log TaskLog, adminID string) (TaskEvent, bool) {
	active := map[string]TaskEvent{}
	for _, event := range log.Events {
		if event.AdminID != adminID {
			continue
		}
		id := normalizeID(event.TaskID)
		switch event.Type {
		case "start":
			active[id] = event
		case "stop":
			delete(active, id)
		}
	}
	if len(active) == 0 {
		return TaskEvent{}, false
	}
	keys := make([]string, 0, len(active))
	for id := range active {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	return active[keys[0]], true
}

func taskIsRunning(log TaskLog, taskID, adminID string) (TaskEvent, bool) {
	taskID = normalizeID(taskID)
	var current TaskEvent
	running := false
	for _, event := range log.Events {
		if normalizeID(event.TaskID) != taskID || event.AdminID != adminID {
			continue
		}
		switch event.Type {
		case "start":
			current = event
			running = true
		case "stop":
			running = false
		}
	}
	return current, running
}

func ensureProjectExists(root, id string) error {
	if strings.TrimSpace(id) == "" {
		return nil
	}
	id = normalizeID(id)
	files, err := listMarkdownFiles(root, "proyectos")
	if err != nil {
		return err
	}
	for _, path := range files {
		meta, err := parseMarkdown(path)
		if err != nil {
			return err
		}
		if normalizeID(meta.ExtraString("id")) == id {
			return nil
		}
	}
	return NewError(ExitValidation, "project_not_found", "la tarea referencia un proyecto inexistente", id)
}

func generatedStateExists(root string) bool {
	_, err := os.Stat(filepath.Join(root, "data", "generated", "site_state.json"))
	return err == nil
}

func statusIsClosed(status string) bool {
	switch strings.ToLower(status) {
	case "done", "completed", "cancelled", "archived":
		return true
	default:
		return false
	}
}

func pageIDDuplicates(records []string) []string {
	counts := map[string]int{}
	for _, id := range records {
		counts[id]++
	}
	out := []string{}
	for id, count := range counts {
		if id != "" && count > 1 {
			out = append(out, fmt.Sprintf("%s (%d)", id, count))
		}
	}
	sort.Strings(out)
	return out
}
