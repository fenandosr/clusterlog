package clusterlog

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

func (a *App) runTask(root string, args []string) error {
	if len(args) == 0 {
		return NewError(ExitUsage, "task_usage", "uso: task create | start ID | stop ID | adjust ID | status ID | done ID | list", nil)
	}
	switch args[0] {
	case "create":
		return a.runTaskCreate(root, args[1:])
	case "start":
		return a.runTaskStart(root, args[1:])
	case "stop":
		return a.runTaskStop(root, args[1:])
	case "adjust":
		return a.runTaskAdjust(root, args[1:])
	case "status":
		return a.runTaskStatus(root, args[1:])
	case "done":
		return a.runTaskDone(root, args[1:])
	case "list":
		return a.runTaskList(root, args[1:])
	default:
		return NewError(ExitUsage, "task_usage", "uso: task create | start ID | stop ID | adjust ID | status ID | done ID | list", args[0])
	}
}

func (a *App) runTaskCreate(root string, args []string) error {
	fs := newFlagSet("task create")
	title := fs.String("title", "", "título")
	description := fs.String("description", "", "resumen")
	projectID := fs.String("project", "", "ID de proyecto")
	priority := fs.String("priority", "", "low|medium|high|urgent")
	assignees := fs.String("assignees", "", "administradores separados por coma")
	estimate := fs.String("estimate", "", "minutos, 90m, 2h")
	due := fs.String("due", "", "fecha YYYY-MM-DD")
	tags := fs.String("tags", "", "tags separados por coma")
	systems := fs.String("systems", "", "sistemas separados por coma")
	objective := fs.String("objective", "", "objetivo")
	acceptance := fs.String("acceptance", "", "criterios separados por coma")
	notes := fs.String("notes", "", "notas")
	bodyFile := fs.String("body-file", "", "Markdown o -")
	fromJSON := fs.String("from-json", "", "JSON o -")
	if err := fs.Parse(args); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}
	var input TaskInput
	if *fromJSON != "" {
		if err := decodeJSONInput(*fromJSON, a.Options.Stdin, &input); err != nil {
			return err
		}
	}
	if wasSet(fs, "title") {
		input.Title = *title
	}
	if wasSet(fs, "description") {
		input.Description = *description
	}
	if wasSet(fs, "project") {
		input.ProjectID = *projectID
	}
	if wasSet(fs, "priority") {
		input.Priority = *priority
	}
	if wasSet(fs, "assignees") {
		input.Assignees = splitCSV(*assignees)
	}
	if wasSet(fs, "due") {
		input.Due = *due
	}
	if wasSet(fs, "tags") {
		input.Tags = splitCSV(*tags)
	}
	if wasSet(fs, "systems") {
		input.Systems = splitCSV(*systems)
	}
	if wasSet(fs, "objective") {
		input.Objective = *objective
	}
	if wasSet(fs, "acceptance") {
		input.Acceptance = splitCSV(*acceptance)
	}
	if wasSet(fs, "notes") {
		input.Notes = *notes
	}
	if *estimate != "" {
		minutes, err := parseDurationMinutes(*estimate)
		if err != nil {
			return NewError(ExitUsage, "invalid_estimate", err.Error(), nil)
		}
		input.EstimateMinutes = minutes
	}
	if *bodyFile != "" {
		data, err := readAll(*bodyFile, a.Options.Stdin)
		if err != nil {
			return err
		}
		input.BodyMarkdown = string(data)
	}
	identity, err := a.ResolveIdentity(root)
	if err != nil {
		return err
	}
	result, warnings, err := a.createTask(root, identity, input)
	if err != nil {
		return err
	}
	if a.Options.DryRun {
		return a.printResult("task.create", result, warnings...)
	}
	state, changed, statePath, err := a.Sync(root, false)
	if err != nil {
		return err
	}
	createdPath := filepath.FromSlash(result["path"].(string))
	paths := []string{createdPath}
	if changed {
		paths = append(paths, statePath)
	}
	sha, err := a.commitPaths(root, identity, fmt.Sprintf("task(%s): %s", result["task_id"], input.Title), paths)
	if err != nil {
		return err
	}
	result["commit"] = sha
	result["open_tasks"] = state.Stats.OpenTasks
	return a.printResult("task.create", result, warnings...)
}

func (a *App) runTaskStart(root string, args []string) error {
	if len(args) < 1 {
		return NewError(ExitUsage, "missing_task_id", "uso: task start ID [--note TEXTO]", nil)
	}
	taskID := normalizeID(args[0])
	fs := newFlagSet("task start")
	note := fs.String("note", "", "nota breve")
	if err := fs.Parse(args[1:]); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}
	identity, err := a.ResolveIdentity(root)
	if err != nil {
		return err
	}
	task, _, err := findTask(root, taskID)
	if err != nil {
		return err
	}
	state, err := a.BuildState(root)
	if err != nil {
		return err
	}
	currentStatus := task.Status
	for _, current := range state.Tasks {
		if current.TaskID == task.TaskID {
			currentStatus = current.Status
			break
		}
	}
	if statusIsClosed(currentStatus) {
		return NewError(ExitConflict, "task_closed", "no se puede iniciar una tarea cerrada", map[string]any{
			"task_id": task.TaskID, "status": currentStatus,
		})
	}
	log, err := loadTaskLog(root)
	if err != nil {
		return err
	}
	if active, ok := taskActiveStart(log, identity.Admin.ID); ok {
		if normalizeID(active.TaskID) == task.TaskID {
			return a.printResult("task.start", map[string]any{
				"task_id": task.TaskID, "admin_id": identity.Admin.ID,
				"already_running": true, "started_at": active.At,
			})
		}
		return NewError(ExitConflict, "timer_already_running", "el administrador ya tiene otra tarea activa", map[string]any{
			"active_task_id": active.TaskID,
			"started_at":     active.At,
			"hint":           "ejecute task stop para la tarea activa",
		})
	}
	eventID, err := generateID("EVT", a.Options.Now())
	if err != nil {
		return err
	}
	event := TaskEvent{
		ID: eventID, TaskID: task.TaskID, Type: "start", AdminID: identity.Admin.ID,
		Fingerprint: identity.Fingerprint, At: a.Options.Now().UTC().Format(time.RFC3339Nano), Note: strings.TrimSpace(*note),
	}
	if a.Options.DryRun {
		return a.printResult("task.start", map[string]any{"event": event, "dry_run": true})
	}
	return a.persistTaskEvent(root, identity, event, "start")
}

func (a *App) runTaskStop(root string, args []string) error {
	if len(args) < 1 {
		return NewError(ExitUsage, "missing_task_id", "uso: task stop ID [--note TEXTO]", nil)
	}
	taskID := normalizeID(args[0])
	fs := newFlagSet("task stop")
	note := fs.String("note", "", "nota breve")
	if err := fs.Parse(args[1:]); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}
	identity, err := a.ResolveIdentity(root)
	if err != nil {
		return err
	}
	task, _, err := findTask(root, taskID)
	if err != nil {
		return err
	}
	log, err := loadTaskLog(root)
	if err != nil {
		return err
	}
	started, running := taskIsRunning(log, task.TaskID, identity.Admin.ID)
	if !running {
		return NewError(ExitConflict, "timer_not_running", "no hay un temporizador activo para esta tarea y administrador", map[string]any{
			"task_id": task.TaskID, "admin_id": identity.Admin.ID,
		})
	}
	startedAt, err := time.Parse(time.RFC3339, started.At)
	if err != nil {
		return NewError(ExitValidation, "invalid_start_event", "el evento start tiene una fecha inválida", started)
	}
	now := a.Options.Now().UTC()
	if now.Before(startedAt) {
		return NewError(ExitValidation, "clock_before_start", "la hora actual es anterior al inicio registrado", started)
	}
	elapsed := now.Sub(startedAt)
	minutes := int((elapsed + time.Minute - 1) / time.Minute)
	if minutes < 1 {
		minutes = 1
	}
	eventID, err := generateID("EVT", now)
	if err != nil {
		return err
	}
	event := TaskEvent{
		ID: eventID, TaskID: task.TaskID, Type: "stop", AdminID: identity.Admin.ID,
		Fingerprint: identity.Fingerprint, At: now.Format(time.RFC3339Nano), Minutes: minutes, Note: strings.TrimSpace(*note),
	}
	if a.Options.DryRun {
		return a.printResult("task.stop", map[string]any{"event": event, "elapsed": formatMinutes(minutes), "dry_run": true})
	}
	return a.persistTaskEvent(root, identity, event, "stop")
}

func (a *App) runTaskAdjust(root string, args []string) error {
	if len(args) < 1 {
		return NewError(ExitUsage, "missing_task_id", "uso: task adjust ID --minutes N [--note TEXTO]", nil)
	}
	taskID := normalizeID(args[0])
	fs := newFlagSet("task adjust")
	minutes := fs.Int("minutes", 0, "minutos positivos o negativos")
	note := fs.String("note", "", "justificación del ajuste")
	if err := fs.Parse(args[1:]); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}
	if !wasSet(fs, "minutes") || *minutes == 0 {
		return NewError(ExitUsage, "invalid_adjustment", "--minutes debe ser distinto de cero", nil)
	}
	identity, err := a.ResolveIdentity(root)
	if err != nil {
		return err
	}
	task, _, err := findTask(root, taskID)
	if err != nil {
		return err
	}
	state, err := a.BuildState(root)
	if err != nil {
		return err
	}
	for _, current := range state.Tasks {
		if current.TaskID == task.TaskID && current.ActualMinutes+*minutes < 0 {
			return NewError(ExitValidation, "negative_actual_time", "el ajuste dejaría el tiempo real por debajo de cero", map[string]any{
				"task_id": task.TaskID, "actual_minutes": current.ActualMinutes, "adjustment": *minutes,
			})
		}
	}
	now := a.Options.Now().UTC()
	eventID, err := generateID("EVT", now)
	if err != nil {
		return err
	}
	event := TaskEvent{
		ID: eventID, TaskID: task.TaskID, Type: "adjust", AdminID: identity.Admin.ID,
		Fingerprint: identity.Fingerprint, At: now.Format(time.RFC3339Nano), Minutes: *minutes, Note: strings.TrimSpace(*note),
	}
	if a.Options.DryRun {
		return a.printResult("task.adjust", map[string]any{"event": event, "dry_run": true})
	}
	return a.persistTaskEvent(root, identity, event, "adjust")
}

func (a *App) runTaskStatus(root string, args []string) error {
	if len(args) < 1 {
		return NewError(ExitUsage, "missing_task_id", "uso: task status ID --status backlog|todo|in_progress|blocked|cancelled [--note TEXTO]", nil)
	}
	taskID := normalizeID(args[0])
	fs := newFlagSet("task status")
	status := fs.String("status", "", "backlog|todo|in_progress|blocked|cancelled")
	note := fs.String("note", "", "nota breve")
	if err := fs.Parse(args[1:]); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}
	newStatus := strings.ToLower(strings.TrimSpace(*status))
	allowed := map[string]bool{"backlog": true, "todo": true, "in_progress": true, "blocked": true, "cancelled": true}
	if !allowed[newStatus] {
		return NewError(ExitUsage, "invalid_task_status", "use backlog, todo, in_progress, blocked o cancelled; para completar use task done", newStatus)
	}
	identity, err := a.ResolveIdentity(root)
	if err != nil {
		return err
	}
	task, _, err := findTask(root, taskID)
	if err != nil {
		return err
	}
	state, err := a.BuildState(root)
	if err != nil {
		return err
	}
	for _, current := range state.Tasks {
		if current.TaskID != task.TaskID {
			continue
		}
		if current.Status == newStatus {
			return a.printResult("task.status", map[string]any{"task_id": task.TaskID, "status": newStatus, "already_set": true})
		}
		if statusIsClosed(newStatus) && len(current.RunningBy) > 0 {
			return NewError(ExitConflict, "task_has_running_timers", "detenga todos los temporizadores antes de cerrar la tarea", map[string]any{
				"task_id": task.TaskID, "running_by": current.RunningBy,
			})
		}
	}
	now := a.Options.Now().UTC()
	eventID, err := generateID("EVT", now)
	if err != nil {
		return err
	}
	event := TaskEvent{
		ID: eventID, TaskID: task.TaskID, Type: "status", AdminID: identity.Admin.ID,
		Fingerprint: identity.Fingerprint, At: now.Format(time.RFC3339Nano), Status: newStatus, Note: strings.TrimSpace(*note),
	}
	if a.Options.DryRun {
		return a.printResult("task.status", map[string]any{"event": event, "dry_run": true})
	}
	return a.persistTaskEvent(root, identity, event, "status")
}

func (a *App) runTaskDone(root string, args []string) error {
	if len(args) < 1 {
		return NewError(ExitUsage, "missing_task_id", "uso: task done ID [--note TEXTO]", nil)
	}
	taskID := normalizeID(args[0])
	fs := newFlagSet("task done")
	note := fs.String("note", "", "nota breve")
	if err := fs.Parse(args[1:]); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}
	identity, err := a.ResolveIdentity(root)
	if err != nil {
		return err
	}
	task, _, err := findTask(root, taskID)
	if err != nil {
		return err
	}
	stateBefore, err := a.BuildState(root)
	if err != nil {
		return err
	}
	for _, current := range stateBefore.Tasks {
		if current.TaskID != task.TaskID {
			continue
		}
		if current.Status == "done" {
			return a.printResult("task.done", map[string]any{"task_id": task.TaskID, "already_done": true})
		}
		otherRunners := []string{}
		for _, adminID := range current.RunningBy {
			if adminID != identity.Admin.ID {
				otherRunners = append(otherRunners, adminID)
			}
		}
		if len(otherRunners) > 0 {
			return NewError(ExitConflict, "task_has_other_running_timers", "otros administradores aún registran tiempo en esta tarea", map[string]any{
				"task_id": task.TaskID, "running_by": otherRunners,
			})
		}
	}
	log, err := loadTaskLog(root)
	if err != nil {
		return err
	}
	now := a.Options.Now().UTC()
	newEvents := []TaskEvent{}
	if started, running := taskIsRunning(log, task.TaskID, identity.Admin.ID); running {
		startedAt, parseErr := time.Parse(time.RFC3339, started.At)
		if parseErr != nil {
			return NewError(ExitValidation, "invalid_start_event", "el evento start tiene una fecha inválida", started)
		}
		if now.Before(startedAt) {
			return NewError(ExitValidation, "clock_before_start", "la hora actual es anterior al inicio registrado", started)
		}
		minutes := int((now.Sub(startedAt) + time.Minute - 1) / time.Minute)
		if minutes < 1 {
			minutes = 1
		}
		id, idErr := generateID("EVT", now)
		if idErr != nil {
			return idErr
		}
		newEvents = append(newEvents, TaskEvent{
			ID: id, TaskID: task.TaskID, Type: "stop", AdminID: identity.Admin.ID,
			Fingerprint: identity.Fingerprint, At: now.Format(time.RFC3339Nano), Minutes: minutes,
			Note: "temporizador cerrado al completar la tarea",
		})
	}
	id, err := generateID("EVT", now)
	if err != nil {
		return err
	}
	statusEvent := TaskEvent{
		ID: id, TaskID: task.TaskID, Type: "status", AdminID: identity.Admin.ID,
		Fingerprint: identity.Fingerprint, At: now.Format(time.RFC3339Nano), Status: "done", Note: strings.TrimSpace(*note),
	}
	newEvents = append(newEvents, statusEvent)
	if a.Options.DryRun {
		return a.printResult("task.done", map[string]any{"events": newEvents, "dry_run": true})
	}
	eventPaths, err := writeTaskEvents(root, newEvents)
	if err != nil {
		return err
	}
	state, changed, statePath, err := a.Sync(root, false)
	if err != nil {
		return err
	}
	paths := append([]string(nil), eventPaths...)
	if changed {
		paths = append(paths, statePath)
	}
	sha, err := a.commitPaths(root, identity, fmt.Sprintf("task(%s): done", task.TaskID), paths)
	if err != nil {
		return err
	}
	var final TaskState
	for _, item := range state.Tasks {
		if item.TaskID == task.TaskID {
			final = item
			break
		}
	}
	return a.printResult("task.done", map[string]any{"task": final, "events": newEvents, "commit": sha})
}

func (a *App) runTaskList(root string, args []string) error {
	fs := newFlagSet("task list")
	status := fs.String("status", "", "filtrar por estado")
	mine := fs.Bool("mine", false, "sólo tareas asignadas o ejecutadas por el administrador actual")
	projectID := fs.String("project", "", "filtrar por proyecto")
	if err := fs.Parse(args); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}
	state, err := a.BuildState(root)
	if err != nil {
		return err
	}
	var adminID string
	if *mine {
		identity, err := a.ResolveIdentity(root)
		if err != nil {
			return err
		}
		adminID = identity.Admin.ID
	}
	items := []TaskState{}
	for _, item := range state.Tasks {
		if *status != "" && item.Status != strings.ToLower(*status) {
			continue
		}
		if *projectID != "" && item.ProjectID != normalizeID(*projectID) {
			continue
		}
		if *mine && !contains(item.Assignees, adminID) && !contains(item.RunningBy, adminID) {
			continue
		}
		items = append(items, item)
	}
	return a.printResult("task.list", map[string]any{"count": len(items), "items": items})
}

func (a *App) persistTaskEvent(root string, identity Identity, event TaskEvent, action string) error {
	eventPaths, err := writeTaskEvents(root, []TaskEvent{event})
	if err != nil {
		return err
	}
	state, changed, statePath, err := a.Sync(root, false)
	if err != nil {
		return err
	}
	paths := append([]string(nil), eventPaths...)
	if changed {
		paths = append(paths, statePath)
	}
	sha, err := a.commitPaths(root, identity, fmt.Sprintf("task(%s): %s", event.TaskID, action), paths)
	if err != nil {
		return err
	}
	var task TaskState
	for _, item := range state.Tasks {
		if item.TaskID == normalizeID(event.TaskID) {
			task = item
			break
		}
	}
	return a.printResult("task."+action, map[string]any{"event": event, "task": task, "commit": sha})
}
