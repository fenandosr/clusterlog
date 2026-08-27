package clusterlog

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		return NewError(ExitUsage, "missing_command", usageText(a.Options.Executable), nil)
	}
	command := args[0]
	if command == "help" || command == "--help" || command == "-h" {
		return a.printResult("help", usageText(a.Options.Executable))
	}
	if command == "version" {
		return a.printResult("version", map[string]any{"name": "clusterlog", "version": Version})
	}
	if command == "schema" {
		return a.runSchema(args[1:])
	}
	if command == "bootstrap-instance" {
		return a.runBootstrapInstance(args[1:])
	}

	root, err := a.ResolveRoot()
	if err != nil {
		return err
	}
	switch command {
	case "identity":
		return a.runIdentity(root, args[1:])
	case "admin":
		return a.runAdmin(root, args[1:])
	case "memory":
		return a.runMemory(root, args[1:])
	case "review":
		return a.runReview(root, args[1:])
	case "task":
		return a.runTask(root, args[1:])
	case "project":
		return a.runProject(root, args[1:])
	case "content":
		return a.runContent(root, args[1:])
	case "templates":
		return a.runTemplates(root, args[1:])
	case "sync":
		return a.runSync(root, args[1:])
	case "validate":
		return a.runValidate(root, args[1:])
	case "verify":
		return a.runVerify(root, args[1:])
	default:
		return NewError(ExitUsage, "unknown_command", "comando desconocido: "+command, usageText(a.Options.Executable))
	}
}

func usageText(executable string) string {
	return fmt.Sprintf(`%s [opciones globales] <comando>

Opciones globales (antes del comando):
  --root DIR       raíz del proyecto
  --json           salida estable para agentes
  --ssh-key PATH   llave pública SSH o ruta base
  --commit         crea un commit Git SSH firmado con sólo los archivos modificados
  --dry-run        valida y muestra el resultado sin escribir

Comandos:
  version
  bootstrap-instance --output DIR [--product-name NOMBRE] [--base-url URL]
  identity
  admin add | admin list
  memory create
  review list | review mark ID
  task create | start ID | stop ID | adjust ID | status ID | done ID | list
  project create | list
  content create
  templates extract [--output DIR]
  sync [--check]
  validate
  verify commit [SHA]
  schema [memory|task|project|content]
`, executable)
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

func wasSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func (a *App) runIdentity(root string, args []string) error {
	if len(args) != 0 {
		return NewError(ExitUsage, "unexpected_arguments", "identity no acepta argumentos", args)
	}
	identity, err := a.ResolveIdentity(root)
	if err != nil {
		return err
	}
	return a.printResult("identity", identity)
}

func (a *App) runAdmin(root string, args []string) error {
	if len(args) == 0 {
		return NewError(ExitUsage, "admin_usage", "uso: admin add ... | admin list [--active]", nil)
	}
	if args[0] == "list" {
		fs := newFlagSet("admin list")
		activeOnly := fs.Bool("active", false, "muestra sólo administradores activos")
		if err := fs.Parse(args[1:]); err != nil {
			return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
		}
		registry, err := loadAdmins(root)
		if err != nil {
			return err
		}
		admins := []Admin{}
		for _, admin := range registry.Admins {
			if !*activeOnly || admin.Active {
				admins = append(admins, admin)
			}
		}
		return a.printResult("admin.list", map[string]any{
			"count": len(admins), "admins": admins, "review_policy": registry.ReviewPolicy,
		})
	}
	if args[0] != "add" {
		return NewError(ExitUsage, "admin_usage", "uso: admin add ... | admin list [--active]", args[0])
	}
	fs := newFlagSet("admin add")
	id := fs.String("id", "", "identificador estable")
	name := fs.String("name", "", "nombre")
	email := fs.String("email", "", "correo")
	keyPath := fs.String("key", "", "llave pública")
	rolesCSV := fs.String("roles", "admin", "roles separados por coma")
	if err := fs.Parse(args[1:]); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}
	adminID := slugify(*id)
	if adminID == "" || strings.TrimSpace(*name) == "" || !strings.Contains(*email, "@") {
		return NewError(ExitUsage, "invalid_admin", "id, name y un email válido son obligatorios", nil)
	}
	path := firstNonEmpty(*keyPath, a.Options.SSHKey, os.Getenv("CLUSTERLOG_SSH_KEY"))
	if path == "" {
		return NewError(ExitUsage, "missing_ssh_key", "indique --key o --ssh-key para registrar la llave pública", nil)
	}
	canonical, fingerprint, publicPath, signingPath, err := readPublicKey(path)
	if err != nil {
		return NewError(ExitIdentity, "invalid_ssh_key", err.Error(), path)
	}
	registry, err := loadAdmins(root)
	if err != nil {
		return err
	}
	var actor Identity
	if len(registry.Admins) > 0 {
		actor, err = a.ResolveIdentity(root)
		if err != nil {
			return err
		}
		if !hasRole(actor.Admin, "owner") {
			return NewError(ExitIdentity, "not_authorized", "sólo un administrador con rol owner puede agregar administradores", actor.Admin.ID)
		}
	}
	for _, admin := range registry.Admins {
		if strings.EqualFold(admin.ID, adminID) || strings.EqualFold(admin.Email, *email) {
			return NewError(ExitConflict, "admin_exists", "ya existe un administrador con ese id o correo", admin.ID)
		}
		for _, key := range admin.SSHKeys {
			if key.Fingerprint == fingerprint {
				return NewError(ExitConflict, "key_exists", "la llave SSH ya pertenece a otro administrador", admin.ID)
			}
		}
	}
	roles := splitCSV(*rolesCSV)
	if len(registry.Admins) == 0 && !contains(roles, "owner") {
		roles = append(roles, "owner")
	}
	roles = uniqueSorted(roles)
	newAdmin := Admin{
		ID: adminID, Name: strings.TrimSpace(*name), Email: strings.TrimSpace(*email),
		Active: true, Roles: roles,
		SSHKeys: []SSHKey{{Fingerprint: fingerprint, PublicKey: canonical, AddedAt: a.Options.Now().UTC().Format(time.RFC3339Nano)}},
	}
	registry.Admins = append(registry.Admins, newAdmin)
	sort.Slice(registry.Admins, func(i, j int) bool { return registry.Admins[i].ID < registry.Admins[j].ID })
	if a.Options.DryRun {
		return a.printResult("admin.add", map[string]any{"admin": newAdmin, "dry_run": true})
	}
	if err := saveAdmins(root, registry); err != nil {
		return err
	}
	state, changed, statePath, err := a.Sync(root, false)
	if err != nil {
		return err
	}
	commitIdentity := actor
	if len(registry.Admins) == 1 {
		commitIdentity = Identity{Admin: newAdmin, Fingerprint: fingerprint, PublicKey: canonical, PublicPath: publicPath, SigningKey: signingPath}
	}
	paths := []string{filepath.Join(root, "data", "admins.json"), filepath.Join(root, "config", "allowed_signers")}
	if changed {
		paths = append(paths, statePath)
	}
	sha, err := a.commitPaths(root, commitIdentity, "admin: add "+newAdmin.ID, paths)
	if err != nil {
		return err
	}
	warnings := []string{}
	if !a.Options.Commit {
		warnings = append(warnings, "proteja data/admins.json y exija commits SSH firmados antes de usarlo en producción")
	}
	return a.printResult("admin.add", map[string]any{"admin": newAdmin, "state": state.Stats, "commit": sha}, warnings...)
}

func (a *App) runMemory(root string, args []string) error {
	if len(args) == 0 || args[0] != "create" {
		return NewError(ExitUsage, "memory_usage", "uso: memory create --title ... [--from-json ARCHIVO|-]", nil)
	}
	fs := newFlagSet("memory create")
	title := fs.String("title", "", "título")
	description := fs.String("description", "", "resumen")
	tags := fs.String("tags", "", "tags separados por coma")
	systems := fs.String("systems", "", "sistemas separados por coma")
	risk := fs.String("risk", "", "low|medium|high|critical")
	ticket := fs.String("ticket", "", "ticket relacionado")
	review := fs.Bool("review", true, "requiere revisión")
	reviewPolicy := fs.String("review-policy", "", "all-active|all-active-except-author")
	bodyFile := fs.String("body-file", "", "Markdown o -")
	fromJSON := fs.String("from-json", "", "JSON o -")
	duration := fs.String("duration", "", "minutos, 90m, 2h")
	agent := fs.String("agent", "", "agente que preparó el texto")
	if err := fs.Parse(args[1:]); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}
	var input MemoryInput
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
	if wasSet(fs, "tags") {
		input.Tags = splitCSV(*tags)
	}
	if wasSet(fs, "systems") {
		input.Systems = splitCSV(*systems)
	}
	if wasSet(fs, "risk") {
		input.Risk = *risk
	}
	if wasSet(fs, "ticket") {
		input.Ticket = *ticket
	}
	if wasSet(fs, "review") {
		input.ReviewRequired = review
	}
	if wasSet(fs, "review-policy") {
		input.ReviewPolicy = *reviewPolicy
	}
	if wasSet(fs, "agent") {
		input.Agent = *agent
	}
	if *duration != "" {
		minutes, err := parseDurationMinutes(*duration)
		if err != nil {
			return NewError(ExitUsage, "invalid_duration", err.Error(), nil)
		}
		input.DurationMinutes = minutes
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
	result, warnings, err := a.createMemory(root, identity, input)
	if err != nil {
		return err
	}
	if a.Options.DryRun {
		return a.printResult("memory.create", result, warnings...)
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
	sha, err := a.commitPaths(root, identity, fmt.Sprintf("memory(%s): %s", result["entry_id"], input.Title), paths)
	if err != nil {
		return err
	}
	result["commit"] = sha
	result["pending_reviews"] = state.Stats.PendingReviews
	return a.printResult("memory.create", result, warnings...)
}

func (a *App) runReview(root string, args []string) error {
	if len(args) == 0 {
		return NewError(ExitUsage, "review_usage", "uso: review list | review mark ID [--note TEXTO]", nil)
	}
	switch args[0] {
	case "list":
		fs := newFlagSet("review list")
		mine := fs.Bool("mine", false, "sólo pendientes del administrador actual")
		if err := fs.Parse(args[1:]); err != nil {
			return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
		}
		state, err := a.BuildState(root)
		if err != nil {
			return err
		}
		queue := state.ReviewQueue
		if *mine {
			identity, err := a.ResolveIdentity(root)
			if err != nil {
				return err
			}
			filtered := []ReviewQueueItem{}
			for _, item := range queue {
				if contains(item.MissingReviewers, identity.Admin.ID) {
					filtered = append(filtered, item)
				}
			}
			queue = filtered
		}
		return a.printResult("review.list", map[string]any{"count": len(queue), "items": queue})
	case "mark":
		if len(args) < 2 {
			return NewError(ExitUsage, "missing_entry_id", "indique el ID de la memoria", nil)
		}
		entryID := args[1]
		fs := newFlagSet("review mark")
		note := fs.String("note", "", "nota breve")
		if err := fs.Parse(args[2:]); err != nil {
			return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
		}
		identity, err := a.ResolveIdentity(root)
		if err != nil {
			return err
		}
		memory, _, err := findMemory(root, entryID)
		if err != nil {
			return err
		}
		if !memory.ReviewRequired {
			return NewError(ExitValidation, "review_not_required", "la memoria no requiere revisión", memory.EntryID)
		}
		registry, err := loadAdmins(root)
		if err != nil {
			return err
		}
		if memory.ReviewerSnapshot {
			if !contains(memory.Reviewers, identity.Admin.ID) {
				return NewError(ExitValidation, "review_not_required_for_admin", "el administrador no forma parte del conjunto de revisores capturado para esta memoria", map[string]any{
					"admin_id": identity.Admin.ID, "reviewers": memory.Reviewers,
				})
			}
		} else {
			policy := firstNonEmpty(memory.ReviewPolicy, registry.ReviewPolicy.Default)
			if (policy == "all-active-except-author" || (policy == "all-active" && !registry.ReviewPolicy.IncludeAuthor)) && identity.Admin.ID == memory.AuthorID {
				return NewError(ExitValidation, "review_not_required_for_author", "la política excluye al autor de la revisión", identity.Admin.ID)
			}
		}
		log, err := loadReviewLog(root)
		if err != nil {
			return err
		}
		for _, event := range log.Events {
			if normalizeID(event.EntryID) == memory.EntryID && event.AdminID == identity.Admin.ID && event.ContentSHA256 == memory.ContentSHA256 {
				state, err := a.BuildState(root)
				if err != nil {
					return err
				}
				return a.printResult("review.mark", map[string]any{"entry_id": memory.EntryID, "admin_id": identity.Admin.ID, "already_reviewed": true, "pending_reviews": state.Stats.PendingReviews})
			}
		}
		eventID, err := generateID("REV", a.Options.Now())
		if err != nil {
			return err
		}
		event := ReviewEvent{ID: eventID, EntryID: memory.EntryID, AdminID: identity.Admin.ID, Fingerprint: identity.Fingerprint, ReviewedAt: a.Options.Now().UTC().Format(time.RFC3339Nano), ContentSHA256: memory.ContentSHA256, Note: strings.TrimSpace(*note)}
		if a.Options.DryRun {
			return a.printResult("review.mark", map[string]any{"event": event, "dry_run": true})
		}
		eventPath, err := writeReviewEvent(root, event)
		if err != nil {
			return err
		}
		state, changed, statePath, err := a.Sync(root, false)
		if err != nil {
			return err
		}
		paths := []string{eventPath}
		if changed {
			paths = append(paths, statePath)
		}
		sha, err := a.commitPaths(root, identity, fmt.Sprintf("review(%s): %s", memory.EntryID, identity.Admin.ID), paths)
		if err != nil {
			return err
		}
		return a.printResult("review.mark", map[string]any{"event": event, "pending_reviews": state.Stats.PendingReviews, "commit": sha})
	default:
		return NewError(ExitUsage, "review_usage", "uso: review list | review mark ID", args[0])
	}
}
