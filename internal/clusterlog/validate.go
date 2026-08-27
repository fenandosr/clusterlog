package clusterlog

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ValidationIssue struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type ValidationReport struct {
	Valid       bool              `json:"valid"`
	Errors      []ValidationIssue `json:"errors"`
	Warnings    []ValidationIssue `json:"warnings"`
	Checks      map[string]int    `json:"checks"`
	ZolaChecked bool              `json:"zola_checked"`
}

type pageRecord struct {
	Section string
	Meta    PageMeta
	ID      string
}

func (r *ValidationReport) add(level, code, path, message string, details any) {
	issue := ValidationIssue{Level: level, Code: code, Path: filepath.ToSlash(path), Message: message, Details: details}
	if level == "warning" {
		r.Warnings = append(r.Warnings, issue)
	} else {
		r.Errors = append(r.Errors, issue)
	}
}

func valueStrings(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprint(item))
		}
		return out
	case string:
		if v != "" {
			return []string{v}
		}
	}
	return nil
}

func fingerprintBelongs(registry AdminRegistry, adminID, fingerprint string) bool {
	admin, ok := adminByID(registry, adminID)
	if !ok {
		return false
	}
	for _, key := range admin.SSHKeys {
		if key.Fingerprint == fingerprint {
			return true
		}
	}
	return false
}

func validTaskStatus(status string) bool {
	switch status {
	case "backlog", "todo", "in_progress", "blocked", "done", "cancelled":
		return true
	default:
		return false
	}
}

func validProjectStatus(status string) bool {
	switch status {
	case "planned", "active", "blocked", "done", "completed", "cancelled", "archived":
		return true
	default:
		return false
	}
}

func (a *App) runValidate(root string, args []string) error {
	fs := newFlagSet("validate")
	skipZola := fs.Bool("skip-zola", false, "omite zola check")
	requireZola := fs.Bool("require-zola", false, "falla si Zola no está instalado")
	if err := fs.Parse(args); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}
	report := ValidationReport{Checks: map[string]int{}, Errors: []ValidationIssue{}, Warnings: []ValidationIssue{}}

	registry, err := loadAdmins(root)
	if err != nil {
		report.add("error", "admins_invalid", filepath.Join(root, "data", "admins.json"), err.Error(), nil)
	} else {
		a.validateAdmins(root, registry, &report)
	}

	pages := a.validatePages(root, registry, &report)
	a.validateEvents(root, registry, pages, &report)
	a.validateRunbooks(root, &report)

	if len(report.Errors) == 0 {
		if _, _, _, err := a.Sync(root, true); err != nil {
			report.add("error", "generated_state_stale", filepath.Join(root, "data", "generated", "site_state.json"), err.Error(), nil)
		} else {
			report.Checks["generated_state"] = 1
		}
	}

	if !*skipZola {
		zolaPath, lookupErr := exec.LookPath("zola")
		if lookupErr != nil {
			if *requireZola {
				report.add("error", "zola_missing", "", "Zola no está instalado o no está en PATH", nil)
			} else {
				report.add("warning", "zola_missing", "", "Zola no está instalado; se omitió la validación de plantillas y enlaces", nil)
			}
		} else {
			report.ZolaChecked = true
			cmd := exec.Command(zolaPath, "check")
			cmd.Dir = root
			var output bytes.Buffer
			cmd.Stdout = &output
			cmd.Stderr = &output
			if err := cmd.Run(); err != nil {
				report.add("error", "zola_check_failed", filepath.Join(root, "zola.toml"), "zola check falló", strings.TrimSpace(output.String()))
			} else {
				report.Checks["zola"] = 1
			}
		}
	}

	report.Valid = len(report.Errors) == 0
	if !report.Valid {
		return NewError(ExitValidation, "validation_failed", "la validación del proyecto encontró errores", report)
	}
	warnings := make([]string, 0, len(report.Warnings))
	for _, issue := range report.Warnings {
		warnings = append(warnings, issue.Message)
	}
	return a.printResult("validate", report, warnings...)
}

func (a *App) validateAdmins(root string, registry AdminRegistry, report *ValidationReport) {
	if registry.ReviewPolicy.Default != "all-active" && registry.ReviewPolicy.Default != "all-active-except-author" {
		report.add("error", "invalid_default_review_policy", filepath.Join(root, "data", "admins.json"), "review_policy.default no es válido", registry.ReviewPolicy.Default)
	}
	ids := map[string]string{}
	emails := map[string]string{}
	fingerprints := map[string]string{}
	ownerCount := 0
	for _, admin := range registry.Admins {
		report.Checks["admins"]++
		idKey := normalizeID(admin.ID)
		if admin.ID == "" || slugify(admin.ID) != admin.ID {
			report.add("error", "invalid_admin_id", filepath.Join(root, "data", "admins.json"), "el ID del administrador debe ser un slug estable en minúsculas", admin.ID)
		}
		if prior, ok := ids[idKey]; ok {
			report.add("error", "duplicate_admin_id", filepath.Join(root, "data", "admins.json"), "ID de administrador duplicado", []string{prior, admin.ID})
		}
		ids[idKey] = admin.ID
		emailKey := strings.ToLower(strings.TrimSpace(admin.Email))
		if emailKey == "" || !strings.Contains(emailKey, "@") {
			report.add("error", "invalid_admin_email", filepath.Join(root, "data", "admins.json"), "correo de administrador inválido", admin.ID)
		}
		if prior, ok := emails[emailKey]; ok {
			report.add("error", "duplicate_admin_email", filepath.Join(root, "data", "admins.json"), "correo de administrador duplicado", []string{prior, admin.ID})
		}
		emails[emailKey] = admin.ID
		if hasRole(admin, "owner") && admin.Active {
			ownerCount++
		}
		if admin.Active && len(admin.SSHKeys) == 0 {
			report.add("error", "active_admin_without_key", filepath.Join(root, "data", "admins.json"), "un administrador activo debe tener al menos una llave SSH", admin.ID)
		}
		for _, key := range admin.SSHKeys {
			report.Checks["ssh_keys"]++
			_, computed, err := parsePublicKeyLine(key.PublicKey)
			if err != nil {
				report.add("error", "invalid_public_key", filepath.Join(root, "data", "admins.json"), err.Error(), admin.ID)
				continue
			}
			if key.Fingerprint != computed {
				report.add("error", "fingerprint_mismatch", filepath.Join(root, "data", "admins.json"), "la huella almacenada no corresponde a la llave pública", map[string]any{"admin_id": admin.ID, "stored": key.Fingerprint, "computed": computed})
			}
			if prior, ok := fingerprints[key.Fingerprint]; ok {
				report.add("error", "duplicate_ssh_key", filepath.Join(root, "data", "admins.json"), "una llave SSH está asignada a más de un administrador", []string{prior, admin.ID})
			}
			fingerprints[key.Fingerprint] = admin.ID
		}
	}
	if len(registry.Admins) > 0 && ownerCount == 0 {
		report.add("error", "missing_owner", filepath.Join(root, "data", "admins.json"), "debe existir al menos un administrador activo con rol owner", nil)
	}

	allowedPath := filepath.Join(root, "config", "allowed_signers")
	allowedData, err := os.ReadFile(allowedPath)
	if err != nil {
		report.add("error", "allowed_signers_missing", allowedPath, "no se pudo leer el archivo de firmantes autorizados", err.Error())
		return
	}
	allowed := string(allowedData)
	for _, admin := range registry.Admins {
		if !admin.Active {
			continue
		}
		for _, key := range admin.SSHKeys {
			fields := strings.Fields(key.PublicKey)
			if len(fields) < 2 {
				continue
			}
			expected := fmt.Sprintf("%s namespaces=\"git\" %s %s", admin.Email, fields[0], fields[1])
			if !strings.Contains(allowed, expected) {
				report.add("error", "allowed_signer_missing", allowedPath, "falta una llave activa en allowed_signers", map[string]any{"admin_id": admin.ID, "fingerprint": key.Fingerprint})
			}
		}
	}
}

func (a *App) validatePages(root string, registry AdminRegistry, report *ValidationReport) []pageRecord {
	sections := []string{"documentacion", "manuales", "memorias", "proyectos", "tareas"}
	pages := []pageRecord{}
	ids := []string{}
	for _, section := range sections {
		files, err := listMarkdownFiles(root, section)
		if err != nil {
			report.add("error", "content_walk_failed", filepath.Join(root, "content", section), err.Error(), nil)
			continue
		}
		for _, path := range files {
			report.Checks["content_pages"]++
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				report.add("error", "content_read_failed", path, readErr.Error(), nil)
				continue
			}
			if findings := scanSecrets(string(data)); len(findings) > 0 {
				report.add("error", "possible_secret", path, "el contenido parece incluir secretos", findings)
			}
			meta, parseErr := parseMarkdown(path)
			if parseErr != nil {
				report.add("error", "front_matter_invalid", path, parseErr.Error(), nil)
				continue
			}
			id := normalizeID(meta.ExtraString("id"))
			if id == "" {
				report.add("error", "missing_page_id", path, "la página no tiene extra.id", nil)
			}
			ids = append(ids, id)
			pages = append(pages, pageRecord{Section: section, Meta: meta, ID: id})
			if strings.TrimSpace(meta.String("title")) == "" {
				report.add("error", "missing_title", path, "la página no tiene título", nil)
			}
			authors := valueStrings(meta.Root["authors"])
			for _, author := range authors {
				if _, ok := adminByID(registry, author); !ok && author != "sistema" {
					report.add("error", "unknown_author", path, "la página referencia un autor no registrado", author)
				}
			}
		}
	}
	for _, duplicate := range pageIDDuplicates(ids) {
		report.add("error", "duplicate_page_id", filepath.Join(root, "content"), "hay IDs de contenido duplicados", duplicate)
	}

	projectIDs := map[string]bool{}
	for _, page := range pages {
		if page.Section == "proyectos" {
			projectIDs[page.ID] = true
		}
	}
	for _, page := range pages {
		path := page.Meta.Path
		switch page.Section {
		case "memorias":
			if !strings.HasPrefix(page.ID, "MEM-") {
				report.add("error", "invalid_memory_id", path, "el ID de memoria debe iniciar con MEM-", page.ID)
			}
			authorID := page.Meta.ExtraString("author_id")
			if _, ok := adminByID(registry, authorID); !ok && authorID != "sistema" {
				report.add("error", "unknown_memory_author", path, "author_id no corresponde a un administrador", authorID)
			}
			fingerprint := page.Meta.ExtraString("author_fingerprint")
			if authorID != "sistema" && !fingerprintBelongs(registry, authorID, fingerprint) {
				report.add("error", "memory_fingerprint_mismatch", path, "author_fingerprint no pertenece al autor", map[string]any{"author_id": authorID, "fingerprint": fingerprint})
			}
			policy := page.Meta.ExtraString("review_policy")
			if policy != "" && policy != "all-active" && policy != "all-active-except-author" {
				report.add("error", "invalid_review_policy", path, "review_policy no es válido", policy)
			}
			if page.Meta.ExtraBool("reviewer_snapshot") {
				seenReviewers := map[string]bool{}
				for _, reviewer := range page.Meta.ExtraStrings("reviewers") {
					if seenReviewers[reviewer] {
						report.add("error", "duplicate_reviewer", path, "un revisor aparece más de una vez", reviewer)
					}
					seenReviewers[reviewer] = true
					if _, ok := adminByID(registry, reviewer); !ok {
						report.add("error", "unknown_reviewer", path, "el conjunto capturado referencia un administrador inexistente", reviewer)
					}
				}
			}
		case "tareas":
			if !strings.HasPrefix(page.ID, "TSK-") {
				report.add("error", "invalid_task_id", path, "el ID de tarea debe iniciar con TSK-", page.ID)
			}
			priority := page.Meta.ExtraString("priority")
			if priority != "low" && priority != "medium" && priority != "high" && priority != "urgent" {
				report.add("error", "invalid_task_priority", path, "priority no es válida", priority)
			}
			status := firstNonEmpty(page.Meta.ExtraString("status"), "todo")
			if !validTaskStatus(status) {
				report.add("error", "invalid_task_status", path, "status no es válido", status)
			}
			if due := page.Meta.ExtraString("due"); validateDate(due) != nil {
				report.add("error", "invalid_due", path, "due debe usar YYYY-MM-DD", due)
			}
			projectID := normalizeID(page.Meta.ExtraString("project_id"))
			if projectID != "" && !projectIDs[projectID] {
				report.add("error", "unknown_project", path, "project_id no existe", projectID)
			}
			for _, assignee := range page.Meta.ExtraStrings("assignees") {
				admin, ok := adminByID(registry, assignee)
				if !ok || !admin.Active {
					report.add("error", "invalid_assignee", path, "assignee no existe o está inactivo", assignee)
				}
			}
		case "proyectos":
			if !strings.HasPrefix(page.ID, "PRJ-") {
				report.add("error", "invalid_project_id", path, "el ID de proyecto debe iniciar con PRJ-", page.ID)
			}
			status := firstNonEmpty(page.Meta.ExtraString("status"), "planned")
			if !validProjectStatus(status) {
				report.add("error", "invalid_project_status", path, "status no es válido", status)
			}
			if target := page.Meta.ExtraString("target_date"); validateDate(target) != nil {
				report.add("error", "invalid_target_date", path, "target_date debe usar YYYY-MM-DD", target)
			}
			for _, owner := range page.Meta.ExtraStrings("owners") {
				admin, ok := adminByID(registry, owner)
				if !ok || !admin.Active {
					report.add("error", "invalid_owner", path, "owner no existe o está inactivo", owner)
				}
			}
		case "documentacion":
			if !strings.HasPrefix(page.ID, "DOC-") {
				report.add("error", "invalid_document_id", path, "el ID de documentación debe iniciar con DOC-", page.ID)
			}
		case "manuales":
			if !strings.HasPrefix(page.ID, "RUN-") {
				report.add("error", "invalid_manual_id", path, "el ID de manual debe iniciar con RUN-", page.ID)
			}
		}
	}
	return pages
}

func (a *App) validateEvents(root string, registry AdminRegistry, pages []pageRecord, report *ValidationReport) {
	memories := map[string]pageRecord{}
	tasks := map[string]pageRecord{}
	for _, page := range pages {
		if page.Section == "memorias" {
			memories[page.ID] = page
		}
		if page.Section == "tareas" {
			tasks[page.ID] = page
		}
	}
	reviews, err := loadReviewLog(root)
	if err != nil {
		report.add("error", "review_log_invalid", filepath.Join(root, "data", reviewEventsDirName), err.Error(), nil)
	} else {
		seen := map[string]bool{}
		for _, event := range reviews.Events {
			report.Checks["review_events"]++
			if seen[event.ID] {
				report.add("error", "duplicate_review_event", filepath.Join(root, "data", reviewEventsDirName), "ID de evento de revisión duplicado", event.ID)
			}
			seen[event.ID] = true
			if _, ok := memories[normalizeID(event.EntryID)]; !ok {
				report.add("warning", "orphan_review_event", filepath.Join(root, "data", reviewEventsDirName), "el evento apunta a una memoria que ya no está en el árbol actual", event.EntryID)
			}
			if _, ok := adminByID(registry, event.AdminID); !ok {
				report.add("error", "unknown_review_admin", filepath.Join(root, "data", reviewEventsDirName), "admin_id de revisión no existe", event.AdminID)
			}
			if !fingerprintBelongs(registry, event.AdminID, event.Fingerprint) {
				report.add("error", "review_fingerprint_mismatch", filepath.Join(root, "data", reviewEventsDirName), "la huella del evento no pertenece al administrador", event.ID)
			}
			if _, parseErr := time.Parse(time.RFC3339, event.ReviewedAt); parseErr != nil {
				report.add("error", "invalid_review_time", filepath.Join(root, "data", reviewEventsDirName), "reviewed_at no es RFC3339", event.ID)
			}
			if len(event.ContentSHA256) != 64 {
				report.add("error", "invalid_review_hash", filepath.Join(root, "data", reviewEventsDirName), "content_sha256 no tiene longitud SHA-256", event.ID)
			}
		}
	}

	log, err := loadTaskLog(root)
	if err != nil {
		report.add("error", "task_log_invalid", filepath.Join(root, "data", taskEventsDirName), err.Error(), nil)
		return
	}
	seen := map[string]bool{}
	activeTimer := map[string]TaskEvent{}
	totalMinutes := map[string]int{}
	for _, event := range log.Events {
		report.Checks["task_events"]++
		if seen[event.ID] {
			report.add("error", "duplicate_task_event", filepath.Join(root, "data", taskEventsDirName), "ID de evento de tarea duplicado", event.ID)
		}
		seen[event.ID] = true
		if _, ok := tasks[normalizeID(event.TaskID)]; !ok {
			report.add("error", "unknown_task_event_target", filepath.Join(root, "data", taskEventsDirName), "el evento apunta a una tarea inexistente", event.TaskID)
		}
		if _, ok := adminByID(registry, event.AdminID); !ok {
			report.add("error", "unknown_task_admin", filepath.Join(root, "data", taskEventsDirName), "admin_id de evento no existe", event.AdminID)
		}
		if !fingerprintBelongs(registry, event.AdminID, event.Fingerprint) {
			report.add("error", "task_fingerprint_mismatch", filepath.Join(root, "data", taskEventsDirName), "la huella del evento no pertenece al administrador", event.ID)
		}
		if _, parseErr := time.Parse(time.RFC3339, event.At); parseErr != nil {
			report.add("error", "invalid_task_event_time", filepath.Join(root, "data", taskEventsDirName), "at no es RFC3339", event.ID)
		}
		switch event.Type {
		case "start":
			if active, ok := activeTimer[event.AdminID]; ok {
				code := "duplicate_task_start"
				message := "el administrador inició dos veces la misma tarea sin detenerla"
				if normalizeID(active.TaskID) != normalizeID(event.TaskID) {
					code = "concurrent_task_timers"
					message = "el administrador tiene más de un temporizador activo"
				}
				report.add("error", code, filepath.Join(root, "data", taskEventsDirName), message, map[string]any{
					"admin_id": event.AdminID, "active_event": active.ID, "new_event": event.ID,
				})
			}
			activeTimer[event.AdminID] = event
		case "stop":
			active, ok := activeTimer[event.AdminID]
			if !ok || normalizeID(active.TaskID) != normalizeID(event.TaskID) {
				report.add("error", "task_stop_without_start", filepath.Join(root, "data", taskEventsDirName), "el evento stop no corresponde a un temporizador activo", event.ID)
			} else {
				delete(activeTimer, event.AdminID)
			}
			if event.Minutes <= 0 {
				report.add("error", "invalid_stop_minutes", filepath.Join(root, "data", taskEventsDirName), "un evento stop debe registrar minutos positivos", event.ID)
			}
			totalMinutes[normalizeID(event.TaskID)] += event.Minutes
		case "adjust":
			totalMinutes[normalizeID(event.TaskID)] += event.Minutes
			if event.Minutes == 0 {
				report.add("error", "invalid_adjust_minutes", filepath.Join(root, "data", taskEventsDirName), "un ajuste no puede ser cero", event.ID)
			}
		case "status":
			if !validTaskStatus(event.Status) {
				report.add("error", "invalid_status_event", filepath.Join(root, "data", taskEventsDirName), "status no es válido", event.Status)
			}
		default:
			report.add("error", "invalid_task_event_type", filepath.Join(root, "data", taskEventsDirName), "tipo de evento desconocido", event.Type)
		}
	}

	for taskID, minutes := range totalMinutes {
		if minutes < 0 {
			report.add("error", "negative_task_time", filepath.Join(root, "data", taskEventsDirName), "los eventos dejan el tiempo acumulado por debajo de cero", map[string]any{
				"task_id": taskID, "actual_minutes": minutes,
			})
		}
	}

	// La lectura estable de errores facilita comparaciones en CI y por agentes.
	sort.Slice(report.Errors, func(i, j int) bool {
		if report.Errors[i].Path == report.Errors[j].Path {
			return report.Errors[i].Code < report.Errors[j].Code
		}
		return report.Errors[i].Path < report.Errors[j].Path
	})
	sort.Slice(report.Warnings, func(i, j int) bool {
		if report.Warnings[i].Path == report.Warnings[j].Path {
			return report.Warnings[i].Code < report.Warnings[j].Code
		}
		return report.Warnings[i].Path < report.Warnings[j].Path
	})
}
