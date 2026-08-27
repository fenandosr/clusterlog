package clusterlog

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type MemoryInput struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Tags            []string `json:"tags"`
	Systems         []string `json:"systems"`
	Risk            string   `json:"risk"`
	Ticket          string   `json:"ticket,omitempty"`
	ReviewRequired  *bool    `json:"review_required,omitempty"`
	ReviewPolicy    string   `json:"review_policy,omitempty"`
	BodyMarkdown    string   `json:"body_markdown,omitempty"`
	Context         string   `json:"context,omitempty"`
	Changes         []string `json:"changes,omitempty"`
	Commands        []string `json:"commands,omitempty"`
	Validation      []string `json:"validation,omitempty"`
	Rollback        string   `json:"rollback,omitempty"`
	StartedAt       string   `json:"started_at,omitempty"`
	FinishedAt      string   `json:"finished_at,omitempty"`
	DurationMinutes int      `json:"duration_minutes,omitempty"`
	Agent           string   `json:"agent,omitempty"`
}

type TaskInput struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	ProjectID       string   `json:"project_id,omitempty"`
	Priority        string   `json:"priority,omitempty"`
	Assignees       []string `json:"assignees,omitempty"`
	EstimateMinutes int      `json:"estimate_minutes,omitempty"`
	Due             string   `json:"due,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	Systems         []string `json:"systems,omitempty"`
	BodyMarkdown    string   `json:"body_markdown,omitempty"`
	Objective       string   `json:"objective,omitempty"`
	Acceptance      []string `json:"acceptance_criteria,omitempty"`
	Notes           string   `json:"notes,omitempty"`
}

type ProjectInput struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Objective    string   `json:"objective"`
	Owners       []string `json:"owners,omitempty"`
	TargetDate   string   `json:"target_date,omitempty"`
	Status       string   `json:"status,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Systems      []string `json:"systems,omitempty"`
	BodyMarkdown string   `json:"body_markdown,omitempty"`
	Scope        []string `json:"scope,omitempty"`
	OutOfScope   []string `json:"out_of_scope,omitempty"`
	Milestones   []string `json:"milestones,omitempty"`
	Success      []string `json:"success_criteria,omitempty"`
}

type ContentInput struct {
	Section      string   `json:"section"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Tags         []string `json:"tags,omitempty"`
	Systems      []string `json:"systems,omitempty"`
	BodyMarkdown string   `json:"body_markdown"`
	Weight       int      `json:"weight,omitempty"`
}

func decodeJSONInput(path string, stdinReader io.Reader, dst any) error {
	data, err := readAll(path, stdinReader)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return NewError(ExitUsage, "invalid_json", "el documento de entrada no es JSON válido o contiene campos desconocidos", err.Error())
	}
	return nil
}

func renderMemoryBody(input MemoryInput) string {
	if strings.TrimSpace(input.BodyMarkdown) != "" {
		return normalizeMarkdownBody(input.BodyMarkdown)
	}
	var b strings.Builder
	b.WriteString("## Contexto\n\n")
	b.WriteString(firstNonEmpty(input.Context, "_No especificado._"))
	b.WriteString("\n\n## Cambios realizados\n\n")
	b.WriteString(bullets(input.Changes))
	b.WriteString("\n## Comandos relevantes (secretos redactados)\n\n")
	b.WriteString(codeBlock(input.Commands))
	b.WriteString("\n## Validación\n\n")
	b.WriteString(bullets(input.Validation))
	b.WriteString("\n## Riesgo y reversión\n\n")
	if input.Rollback == "" {
		b.WriteString("_No se documentó un procedimiento de reversión._\n")
	} else {
		b.WriteString(input.Rollback + "\n")
	}
	if input.Ticket != "" {
		b.WriteString("\n## Referencias\n\n- Ticket: `" + input.Ticket + "`\n")
	}
	return normalizeMarkdownBody(b.String())
}

func renderTaskBody(input TaskInput) string {
	if strings.TrimSpace(input.BodyMarkdown) != "" {
		return normalizeMarkdownBody(input.BodyMarkdown)
	}
	var b strings.Builder
	b.WriteString("## Objetivo\n\n")
	b.WriteString(firstNonEmpty(input.Objective, input.Description, "_No especificado._"))
	b.WriteString("\n\n## Criterios de aceptación\n\n")
	b.WriteString(bullets(input.Acceptance))
	if input.Notes != "" {
		b.WriteString("\n## Notas\n\n" + input.Notes + "\n")
	}
	return normalizeMarkdownBody(b.String())
}

func renderProjectBody(input ProjectInput) string {
	if strings.TrimSpace(input.BodyMarkdown) != "" {
		return normalizeMarkdownBody(input.BodyMarkdown)
	}
	var b strings.Builder
	b.WriteString("## Objetivo\n\n")
	b.WriteString(firstNonEmpty(input.Objective, input.Description, "_No especificado._"))
	b.WriteString("\n\n## Alcance\n\n")
	b.WriteString(bullets(input.Scope))
	b.WriteString("\n## Fuera de alcance\n\n")
	b.WriteString(bullets(input.OutOfScope))
	b.WriteString("\n## Hitos\n\n")
	b.WriteString(bullets(input.Milestones))
	b.WriteString("\n## Criterios de éxito\n\n")
	b.WriteString(bullets(input.Success))
	return normalizeMarkdownBody(b.String())
}

func (a *App) createMemory(root string, identity Identity, input MemoryInput) (map[string]any, []string, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = firstNonEmpty(input.Description, input.Title)
	input.Risk = strings.ToLower(firstNonEmpty(input.Risk, "low"))
	validRisk := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	if input.Title == "" {
		return nil, nil, NewError(ExitUsage, "missing_title", "--title o title en JSON es obligatorio", nil)
	}
	if !validRisk[input.Risk] {
		return nil, nil, NewError(ExitUsage, "invalid_risk", "risk debe ser low, medium, high o critical", input.Risk)
	}
	reviewRequired := true
	if input.ReviewRequired != nil {
		reviewRequired = *input.ReviewRequired
	}
	registry, err := loadAdmins(root)
	if err != nil {
		return nil, nil, err
	}
	input.ReviewPolicy = firstNonEmpty(input.ReviewPolicy, registry.ReviewPolicy.Default, "all-active")
	if input.ReviewPolicy != "all-active" && input.ReviewPolicy != "all-active-except-author" {
		return nil, nil, NewError(ExitUsage, "invalid_review_policy", "review_policy debe ser all-active o all-active-except-author", input.ReviewPolicy)
	}
	reviewers := []string{}
	if reviewRequired {
		reviewers = requiredReviewers(registry, input.ReviewPolicy, identity.Admin.ID)
	}
	now := a.Options.Now().UTC()
	id, err := generateID("MEM", now)
	if err != nil {
		return nil, nil, err
	}
	slug := slugify(id + "-" + input.Title)
	body := renderMemoryBody(input)
	front := strings.Builder{}
	front.WriteString("+++\n")
	fmt.Fprintf(&front, "title = %s\n", tomlString(input.Title))
	fmt.Fprintf(&front, "description = %s\n", tomlString(input.Description))
	fmt.Fprintf(&front, "date = %s\n", now.Format(time.RFC3339))
	fmt.Fprintf(&front, "authors = %s\n", tomlStringArray([]string{identity.Admin.ID}))
	fmt.Fprintf(&front, "slug = %s\n", tomlString(slug))
	front.WriteString("template = \"memory.html\"\n\n")
	front.WriteString("[taxonomies]\n")
	front.WriteString("categories = [\"memorias\"]\n")
	fmt.Fprintf(&front, "tags = %s\n", tomlStringArray(uniqueSorted(input.Tags)))
	fmt.Fprintf(&front, "systems = %s\n\n", tomlStringArray(uniqueSorted(input.Systems)))
	front.WriteString("[extra]\n")
	fmt.Fprintf(&front, "id = %s\n", tomlString(id))
	front.WriteString("kind = \"memory\"\n")
	fmt.Fprintf(&front, "author_id = %s\n", tomlString(identity.Admin.ID))
	fmt.Fprintf(&front, "author_fingerprint = %s\n", tomlString(identity.Fingerprint))
	fmt.Fprintf(&front, "risk = %s\n", tomlString(input.Risk))
	fmt.Fprintf(&front, "ticket = %s\n", tomlString(input.Ticket))
	fmt.Fprintf(&front, "review_required = %t\n", reviewRequired)
	fmt.Fprintf(&front, "review_policy = %s\n", tomlString(input.ReviewPolicy))
	front.WriteString("reviewer_snapshot = true\n")
	fmt.Fprintf(&front, "reviewers = %s\n", tomlStringArray(reviewers))
	fmt.Fprintf(&front, "started_at = %s\n", tomlString(input.StartedAt))
	fmt.Fprintf(&front, "finished_at = %s\n", tomlString(input.FinishedAt))
	fmt.Fprintf(&front, "duration_minutes = %d\n", input.DurationMinutes)
	fmt.Fprintf(&front, "agent = %s\n", tomlString(input.Agent))
	front.WriteString("+++\n\n")
	content := front.String() + body
	if findings := scanSecrets(content); len(findings) > 0 {
		return nil, nil, NewError(ExitValidation, "possible_secret", "la memoria parece contener secretos; redacte antes de registrarla", findings)
	}
	filename := now.Format("2006-01-02") + "-" + slug + ".md"
	path := filepath.Join(root, "content", "memorias", filename)
	if _, err := os.Stat(path); err == nil {
		return nil, nil, NewError(ExitConflict, "file_exists", "ya existe el archivo de memoria", path)
	}
	if !a.Options.DryRun {
		if err := writeFileAtomic(path, []byte(content), 0o644); err != nil {
			return nil, nil, err
		}
	}
	warnings := []string{}
	if !a.Options.Commit {
		warnings = append(warnings, "la atribución local usa la huella SSH, pero la prueba criptográfica exige un commit firmado; repita con --commit o firme el commit manualmente")
	}
	return map[string]any{
		"entry_id": id, "path": filepath.ToSlash(path), "slug": slug,
		"review_required": reviewRequired, "reviewers": reviewers, "content_sha256": bytesSHA256([]byte(content)),
		"dry_run": a.Options.DryRun,
	}, warnings, nil
}

func (a *App) createTask(root string, identity Identity, input TaskInput) (map[string]any, []string, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = firstNonEmpty(input.Description, input.Title)
	input.Priority = strings.ToLower(firstNonEmpty(input.Priority, "medium"))
	if input.Title == "" {
		return nil, nil, NewError(ExitUsage, "missing_title", "el título de la tarea es obligatorio", nil)
	}
	if !map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}[input.Priority] {
		return nil, nil, NewError(ExitUsage, "invalid_priority", "priority debe ser low, medium, high o urgent", input.Priority)
	}
	if err := validateDate(input.Due); err != nil {
		return nil, nil, NewError(ExitUsage, "invalid_due", err.Error(), nil)
	}
	if err := ensureProjectExists(root, input.ProjectID); err != nil {
		return nil, nil, err
	}
	registry, err := loadAdmins(root)
	if err != nil {
		return nil, nil, err
	}
	for _, assignee := range input.Assignees {
		admin, ok := adminByID(registry, assignee)
		if !ok || !admin.Active {
			return nil, nil, NewError(ExitValidation, "invalid_assignee", "la tarea contiene un responsable inexistente o inactivo", assignee)
		}
	}
	now := a.Options.Now().UTC()
	id, err := generateID("TSK", now)
	if err != nil {
		return nil, nil, err
	}
	slug := slugify(id + "-" + input.Title)
	body := renderTaskBody(input)
	var front strings.Builder
	front.WriteString("+++\n")
	fmt.Fprintf(&front, "title = %s\n", tomlString(input.Title))
	fmt.Fprintf(&front, "description = %s\n", tomlString(input.Description))
	fmt.Fprintf(&front, "date = %s\n", now.Format(time.RFC3339))
	fmt.Fprintf(&front, "authors = %s\n", tomlStringArray([]string{identity.Admin.ID}))
	fmt.Fprintf(&front, "slug = %s\n", tomlString(slug))
	front.WriteString("template = \"task.html\"\n\n")
	front.WriteString("[taxonomies]\n")
	front.WriteString("categories = [\"tareas\"]\n")
	fmt.Fprintf(&front, "tags = %s\n", tomlStringArray(uniqueSorted(input.Tags)))
	fmt.Fprintf(&front, "systems = %s\n\n", tomlStringArray(uniqueSorted(input.Systems)))
	front.WriteString("[extra]\n")
	fmt.Fprintf(&front, "id = %s\n", tomlString(id))
	front.WriteString("kind = \"task\"\nstatus = \"todo\"\n")
	fmt.Fprintf(&front, "priority = %s\n", tomlString(input.Priority))
	fmt.Fprintf(&front, "project_id = %s\n", tomlString(normalizeID(input.ProjectID)))
	fmt.Fprintf(&front, "assignees = %s\n", tomlStringArray(uniqueSorted(input.Assignees)))
	fmt.Fprintf(&front, "estimate_minutes = %d\n", input.EstimateMinutes)
	fmt.Fprintf(&front, "due = %s\n", tomlString(input.Due))
	fmt.Fprintf(&front, "created_by = %s\n", tomlString(identity.Admin.ID))
	fmt.Fprintf(&front, "author_fingerprint = %s\n", tomlString(identity.Fingerprint))
	front.WriteString("+++\n\n")
	content := front.String() + body
	if findings := scanSecrets(content); len(findings) > 0 {
		return nil, nil, NewError(ExitValidation, "possible_secret", "la tarea parece contener secretos; redacte antes de registrarla", findings)
	}
	path := filepath.Join(root, "content", "tareas", now.Format("2006-01-02")+"-"+slug+".md")
	if !a.Options.DryRun {
		if err := writeFileAtomic(path, []byte(content), 0o644); err != nil {
			return nil, nil, err
		}
	}
	warnings := []string{}
	if !a.Options.Commit {
		warnings = append(warnings, "use --commit para dejar la operación en un commit SSH firmado")
	}
	return map[string]any{"task_id": id, "path": filepath.ToSlash(path), "slug": slug, "dry_run": a.Options.DryRun}, warnings, nil
}

func (a *App) createProject(root string, identity Identity, input ProjectInput) (map[string]any, []string, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = firstNonEmpty(input.Description, input.Objective, input.Title)
	input.Status = strings.ToLower(firstNonEmpty(input.Status, "planned"))
	if input.Title == "" || strings.TrimSpace(input.Objective) == "" {
		return nil, nil, NewError(ExitUsage, "missing_project_fields", "title y objective son obligatorios", nil)
	}
	if !map[string]bool{"planned": true, "active": true, "blocked": true, "done": true, "cancelled": true, "archived": true}[input.Status] {
		return nil, nil, NewError(ExitUsage, "invalid_project_status", "status debe ser planned, active, blocked, done, cancelled o archived", input.Status)
	}
	if err := validateDate(input.TargetDate); err != nil {
		return nil, nil, NewError(ExitUsage, "invalid_target_date", err.Error(), nil)
	}
	registry, err := loadAdmins(root)
	if err != nil {
		return nil, nil, err
	}
	for _, owner := range input.Owners {
		admin, ok := adminByID(registry, owner)
		if !ok || !admin.Active {
			return nil, nil, NewError(ExitValidation, "invalid_owner", "el proyecto contiene un responsable inexistente o inactivo", owner)
		}
	}
	if len(input.Owners) == 0 {
		input.Owners = []string{identity.Admin.ID}
	}
	now := a.Options.Now().UTC()
	id, err := generateID("PRJ", now)
	if err != nil {
		return nil, nil, err
	}
	slug := slugify(id + "-" + input.Title)
	body := renderProjectBody(input)
	var front strings.Builder
	front.WriteString("+++\n")
	fmt.Fprintf(&front, "title = %s\n", tomlString(input.Title))
	fmt.Fprintf(&front, "description = %s\n", tomlString(input.Description))
	fmt.Fprintf(&front, "date = %s\n", now.Format(time.RFC3339))
	fmt.Fprintf(&front, "authors = %s\n", tomlStringArray([]string{identity.Admin.ID}))
	fmt.Fprintf(&front, "slug = %s\n", tomlString(slug))
	front.WriteString("template = \"project.html\"\n\n")
	front.WriteString("[taxonomies]\n")
	front.WriteString("categories = [\"proyectos\"]\n")
	fmt.Fprintf(&front, "tags = %s\n", tomlStringArray(uniqueSorted(input.Tags)))
	fmt.Fprintf(&front, "systems = %s\n\n", tomlStringArray(uniqueSorted(input.Systems)))
	front.WriteString("[extra]\n")
	fmt.Fprintf(&front, "id = %s\n", tomlString(id))
	front.WriteString("kind = \"project\"\n")
	fmt.Fprintf(&front, "status = %s\n", tomlString(input.Status))
	fmt.Fprintf(&front, "owners = %s\n", tomlStringArray(uniqueSorted(input.Owners)))
	fmt.Fprintf(&front, "target_date = %s\n", tomlString(input.TargetDate))
	fmt.Fprintf(&front, "created_by = %s\n", tomlString(identity.Admin.ID))
	fmt.Fprintf(&front, "author_fingerprint = %s\n", tomlString(identity.Fingerprint))
	front.WriteString("+++\n\n")
	content := front.String() + body
	if findings := scanSecrets(content); len(findings) > 0 {
		return nil, nil, NewError(ExitValidation, "possible_secret", "el proyecto parece contener secretos; redacte antes de registrarlo", findings)
	}
	path := filepath.Join(root, "content", "proyectos", now.Format("2006-01-02")+"-"+slug+".md")
	if !a.Options.DryRun {
		if err := writeFileAtomic(path, []byte(content), 0o644); err != nil {
			return nil, nil, err
		}
	}
	warnings := []string{}
	if !a.Options.Commit {
		warnings = append(warnings, "use --commit para dejar la operación en un commit SSH firmado")
	}
	return map[string]any{"project_id": id, "path": filepath.ToSlash(path), "slug": slug, "dry_run": a.Options.DryRun}, warnings, nil
}

func (a *App) createContent(root string, identity Identity, input ContentInput) (map[string]any, []string, error) {
	input.Section = strings.ToLower(strings.TrimSpace(input.Section))
	if input.Section != "documentacion" && input.Section != "manuales" {
		return nil, nil, NewError(ExitUsage, "invalid_section", "section debe ser documentacion o manuales", input.Section)
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Description = firstNonEmpty(input.Description, input.Title)
	if input.Title == "" || strings.TrimSpace(input.BodyMarkdown) == "" {
		return nil, nil, NewError(ExitUsage, "missing_content_fields", "title y body_markdown son obligatorios", nil)
	}
	now := a.Options.Now().UTC()
	idPrefix := "DOC"
	if input.Section == "manuales" {
		idPrefix = "RUN"
	}
	id, err := generateID(idPrefix, now)
	if err != nil {
		return nil, nil, err
	}
	slug := slugify(id + "-" + input.Title)
	var front strings.Builder
	front.WriteString("+++\n")
	fmt.Fprintf(&front, "title = %s\n", tomlString(input.Title))
	fmt.Fprintf(&front, "description = %s\n", tomlString(input.Description))
	fmt.Fprintf(&front, "date = %s\n", now.Format(time.RFC3339))
	fmt.Fprintf(&front, "authors = %s\n", tomlStringArray([]string{identity.Admin.ID}))
	fmt.Fprintf(&front, "slug = %s\n", tomlString(slug))
	front.WriteString("template = \"page.html\"\n")
	if input.Weight != 0 {
		fmt.Fprintf(&front, "weight = %d\n", input.Weight)
	}
	front.WriteString("\n[taxonomies]\n")
	fmt.Fprintf(&front, "categories = %s\n", tomlStringArray([]string{input.Section}))
	fmt.Fprintf(&front, "tags = %s\n", tomlStringArray(uniqueSorted(input.Tags)))
	fmt.Fprintf(&front, "systems = %s\n\n", tomlStringArray(uniqueSorted(input.Systems)))
	front.WriteString("[extra]\n")
	fmt.Fprintf(&front, "id = %s\n", tomlString(id))
	fmt.Fprintf(&front, "kind = %s\n", tomlString(input.Section))
	fmt.Fprintf(&front, "author_id = %s\n", tomlString(identity.Admin.ID))
	fmt.Fprintf(&front, "author_fingerprint = %s\n", tomlString(identity.Fingerprint))
	front.WriteString("+++\n\n")
	content := front.String() + normalizeMarkdownBody(input.BodyMarkdown)
	if findings := scanSecrets(content); len(findings) > 0 {
		return nil, nil, NewError(ExitValidation, "possible_secret", "el contenido parece incluir secretos; redacte antes de registrarlo", findings)
	}
	path := filepath.Join(root, "content", input.Section, now.Format("2006-01-02")+"-"+slug+".md")
	if !a.Options.DryRun {
		if err := writeFileAtomic(path, []byte(content), 0o644); err != nil {
			return nil, nil, err
		}
	}
	warnings := []string{}
	if !a.Options.Commit {
		warnings = append(warnings, "use --commit para dejar la operación en un commit SSH firmado")
	}
	return map[string]any{"content_id": id, "section": input.Section, "path": filepath.ToSlash(path), "slug": slug, "dry_run": a.Options.DryRun}, warnings, nil
}
