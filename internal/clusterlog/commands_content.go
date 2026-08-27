package clusterlog

import (
	"fmt"
	"path/filepath"
)

func (a *App) runProject(root string, args []string) error {
	if len(args) == 0 {
		return NewError(ExitUsage, "project_usage", "uso: project create | list", nil)
	}
	switch args[0] {
	case "create":
		fs := newFlagSet("project create")
		title := fs.String("title", "", "título")
		description := fs.String("description", "", "resumen")
		objective := fs.String("objective", "", "objetivo")
		owners := fs.String("owners", "", "administradores separados por coma")
		targetDate := fs.String("target-date", "", "fecha YYYY-MM-DD")
		status := fs.String("status", "", "planned|active|blocked|done|cancelled")
		tags := fs.String("tags", "", "tags separados por coma")
		systems := fs.String("systems", "", "sistemas separados por coma")
		scope := fs.String("scope", "", "elementos de alcance separados por coma")
		outOfScope := fs.String("out-of-scope", "", "elementos fuera de alcance separados por coma")
		milestones := fs.String("milestones", "", "hitos separados por coma")
		success := fs.String("success", "", "criterios de éxito separados por coma")
		bodyFile := fs.String("body-file", "", "Markdown o -")
		fromJSON := fs.String("from-json", "", "JSON o -")
		if err := fs.Parse(args[1:]); err != nil {
			return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
		}
		var input ProjectInput
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
		if wasSet(fs, "objective") {
			input.Objective = *objective
		}
		if wasSet(fs, "owners") {
			input.Owners = splitCSV(*owners)
		}
		if wasSet(fs, "target-date") {
			input.TargetDate = *targetDate
		}
		if wasSet(fs, "status") {
			input.Status = *status
		}
		if wasSet(fs, "tags") {
			input.Tags = splitCSV(*tags)
		}
		if wasSet(fs, "systems") {
			input.Systems = splitCSV(*systems)
		}
		if wasSet(fs, "scope") {
			input.Scope = splitCSV(*scope)
		}
		if wasSet(fs, "out-of-scope") {
			input.OutOfScope = splitCSV(*outOfScope)
		}
		if wasSet(fs, "milestones") {
			input.Milestones = splitCSV(*milestones)
		}
		if wasSet(fs, "success") {
			input.Success = splitCSV(*success)
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
		result, warnings, err := a.createProject(root, identity, input)
		if err != nil {
			return err
		}
		if a.Options.DryRun {
			return a.printResult("project.create", result, warnings...)
		}
		state, changed, statePath, err := a.Sync(root, false)
		if err != nil {
			return err
		}
		paths := []string{filepath.FromSlash(result["path"].(string))}
		if changed {
			paths = append(paths, statePath)
		}
		sha, err := a.commitPaths(root, identity, fmt.Sprintf("project(%s): %s", result["project_id"], input.Title), paths)
		if err != nil {
			return err
		}
		result["commit"] = sha
		result["active_projects"] = state.Stats.ActiveProjects
		return a.printResult("project.create", result, warnings...)
	case "list":
		fs := newFlagSet("project list")
		status := fs.String("status", "", "filtrar por estado")
		if err := fs.Parse(args[1:]); err != nil {
			return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
		}
		state, err := a.BuildState(root)
		if err != nil {
			return err
		}
		items := []ProjectState{}
		for _, item := range state.Projects {
			if *status == "" || item.Status == *status {
				items = append(items, item)
			}
		}
		return a.printResult("project.list", map[string]any{"count": len(items), "items": items})
	default:
		return NewError(ExitUsage, "project_usage", "uso: project create | list", args[0])
	}
}

func (a *App) runContent(root string, args []string) error {
	if len(args) == 0 || args[0] != "create" {
		return NewError(ExitUsage, "content_usage", "uso: content create --section documentacion|manuales --title ... --body-file ...", nil)
	}
	fs := newFlagSet("content create")
	section := fs.String("section", "", "documentacion|manuales")
	title := fs.String("title", "", "título")
	description := fs.String("description", "", "resumen")
	tags := fs.String("tags", "", "tags separados por coma")
	systems := fs.String("systems", "", "sistemas separados por coma")
	bodyFile := fs.String("body-file", "", "Markdown o -")
	weight := fs.Int("weight", 0, "orden opcional")
	fromJSON := fs.String("from-json", "", "JSON o -")
	if err := fs.Parse(args[1:]); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}
	var input ContentInput
	if *fromJSON != "" {
		if err := decodeJSONInput(*fromJSON, a.Options.Stdin, &input); err != nil {
			return err
		}
	}
	if wasSet(fs, "section") {
		input.Section = *section
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
	if wasSet(fs, "weight") {
		input.Weight = *weight
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
	result, warnings, err := a.createContent(root, identity, input)
	if err != nil {
		return err
	}
	if a.Options.DryRun {
		return a.printResult("content.create", result, warnings...)
	}
	createdPath := filepath.FromSlash(result["path"].(string))
	sha, err := a.commitPaths(root, identity, fmt.Sprintf("%s(%s): %s", input.Section, result["content_id"], input.Title), []string{createdPath})
	if err != nil {
		return err
	}
	result["commit"] = sha
	return a.printResult("content.create", result, warnings...)
}

func (a *App) runSync(root string, args []string) error {
	fs := newFlagSet("sync")
	check := fs.Bool("check", false, "no escribe; falla si el estado generado está desactualizado")
	if err := fs.Parse(args); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}
	state, changed, path, err := a.Sync(root, *check)
	if err != nil {
		return err
	}
	return a.printResult("sync", map[string]any{
		"path": filepath.ToSlash(path), "changed": changed, "check": *check,
		"source_sha256": state.SourceSHA256, "stats": state.Stats,
	})
}

func (a *App) runVerify(root string, args []string) error {
	if len(args) == 0 || args[0] != "commit" {
		return NewError(ExitUsage, "verify_usage", "uso: verify commit [SHA]", nil)
	}
	sha := "HEAD"
	if len(args) > 1 {
		sha = args[1]
	}
	if len(args) > 2 {
		return NewError(ExitUsage, "unexpected_arguments", "uso: verify commit [SHA]", args[2:])
	}
	if err := verifyCommit(root, sha); err != nil {
		return err
	}
	return a.printResult("verify.commit", map[string]any{"commit": sha, "valid": true})
}
