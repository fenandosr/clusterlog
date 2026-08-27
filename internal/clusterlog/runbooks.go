package clusterlog

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Este archivo valida los runbooks ejecutables (Runme) que viven en
// RUNBOOK.md y runbooks/**/*.md. No ejecuta Runme ni ninguna celda: sólo
// analiza el Markdown como texto para hacer cumplir las reglas de la
// sección 12 del ADR 0004 antes de que alguien abra el documento.
//
// El análisis es un escáner de líneas con estado (no una única expresión
// regular como defensa completa): la regex sólo reconoce la FORMA de una
// línea de fence, y el contenido de atributos siempre se decodifica con
// encoding/json.

var runmeFenceLine = regexp.MustCompile("^```([A-Za-z0-9_+-]*)\\s*(\\{.*\\})?\\s*$")

type runmeCell struct {
	Path  string
	Line  int
	Lang  string
	Name  string
	Tags  []string
	Attrs map[string]any
	Body  string
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

func hasTagPrefix(tags []string, prefix string) bool {
	for _, t := range tags {
		if strings.HasPrefix(t, prefix) {
			return true
		}
	}
	return false
}

func attrBool(attrs map[string]any, key string) (bool, bool) {
	value, ok := attrs[key]
	if !ok {
		return false, false
	}
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		return strings.EqualFold(v, "true"), true
	default:
		return false, true
	}
}

// scanRunmeCells recorre un documento Markdown y devuelve las celdas que
// declaran atributos Runme con al menos un "name". Los fences sin atributos
// (bloques de código normales) se ignoran para efectos de reglas de Runme,
// pero igual se respeta su apertura/cierre para no desalinear el escaneo.
func scanRunmeCells(path string, report *ValidationReport) []runmeCell {
	data, err := os.ReadFile(path)
	if err != nil {
		report.add("error", "runbook_read_failed", path, err.Error(), nil)
		return nil
	}
	cells := []runmeCell{}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")

	inFence := false
	var current *runmeCell
	var body strings.Builder
	for i, raw := range lines {
		lineNo := i + 1
		trimmed := strings.TrimRight(raw, " \t")
		if !inFence {
			m := runmeFenceLine.FindStringSubmatch(trimmed)
			if m == nil {
				continue
			}
			inFence = true
			lang, attrsRaw := m[1], m[2]
			if attrsRaw == "" {
				current = nil
				continue
			}
			var attrs map[string]any
			if jsonErr := json.Unmarshal([]byte(attrsRaw), &attrs); jsonErr != nil {
				report.add("error", "runme_cell_invalid_attrs", path, "los atributos de la celda no son JSON válido: "+jsonErr.Error(), map[string]any{"line": lineNo})
				current = nil
				continue
			}
			name, _ := attrs["name"].(string)
			if name == "" {
				current = nil
				continue
			}
			tagsRaw, _ := attrs["tag"].(string)
			tags := []string{}
			for _, t := range strings.Split(tagsRaw, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
			current = &runmeCell{Path: path, Line: lineNo, Lang: lang, Name: name, Tags: tags, Attrs: attrs}
			body.Reset()
			continue
		}
		// Dentro de un fence: una línea de cierre es ``` sola en la línea.
		if trimmed == "```" {
			inFence = false
			if current != nil {
				current.Body = body.String()
				cells = append(cells, *current)
				current = nil
			}
			continue
		}
		if current != nil {
			body.WriteString(raw)
			body.WriteString("\n")
		}
	}
	return cells
}

func findRunbookFiles(root string) ([]string, error) {
	paths := []string{}
	rootRunbook := filepath.Join(root, "RUNBOOK.md")
	if info, err := os.Stat(rootRunbook); err == nil && !info.IsDir() {
		paths = append(paths, rootRunbook)
	}
	base := filepath.Join(root, "runbooks")
	walkErr := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if os.IsNotExist(walkErr) {
		walkErr = nil
	}
	sort.Strings(paths)
	return paths, walkErr
}

func makefileTargets(root string) map[string]bool {
	targets := map[string]bool{}
	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		return targets
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	targetLine := regexp.MustCompile(`^([A-Za-z0-9_.-]+)\s*:`)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "\t") {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), ".PHONY:") {
			for _, name := range strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), ".PHONY:")) {
				targets[name] = true
			}
			continue
		}
		if m := targetLine.FindStringSubmatch(line); m != nil && !strings.HasPrefix(m[1], ".") {
			targets[m[1]] = true
		}
	}
	return targets
}

var makeInvocation = regexp.MustCompile(`\bmake\s+([A-Za-z0-9_.-]+)`)
var scriptInvocation = regexp.MustCompile(`\.\/(scripts\/[A-Za-z0-9_.-]+\.sh)`)
var infraToolPattern = regexp.MustCompile(`\b(kubectl|terraform|helm)\b`)

const (
	runmeUnknownAttrWarning = "unknown_runme_attr"
)

var knownRunmeAttrs = map[string]bool{
	"name": true, "tag": true, "interactive": true, "excludeFromRunAll": true,
	"id": true, "background": true, "promptEnv": true, "cwd": true, "category": true,
}

// validateRunbooks aplica las reglas de la sección 12 del ADR 0004 sobre
// RUNBOOK.md y runbooks/**/*.md, y confirma que ninguna celda de Runme se
// haya colado en content/ (donde Zola sí publica).
func (a *App) validateRunbooks(root string, report *ValidationReport) {
	files, err := findRunbookFiles(root)
	if err != nil {
		report.add("error", "runbook_walk_failed", filepath.Join(root, "runbooks"), err.Error(), nil)
	}

	targets := makefileTargets(root)
	names := map[string]string{}
	var allCells []runmeCell

	for _, path := range files {
		report.Checks["runbook_files"]++
		cells := scanRunmeCells(path, report)
		allCells = append(allCells, cells...)
		if findings := scanSecrets(mustReadFile(path)); len(findings) > 0 {
			report.add("error", "possible_secret", path, "el runbook parece incluir secretos", findings)
		}
	}

	for _, cell := range allCells {
		report.Checks["runme_cells"]++
		if prior, ok := names[cell.Name]; ok && prior != cell.Path {
			report.add("error", "duplicate_runme_cell_name", cell.Path, "nombre de celda Runme duplicado en el repositorio", map[string]any{"name": cell.Name, "also_in": prior, "line": cell.Line})
		} else {
			names[cell.Name] = cell.Path
		}

		hasRole := hasTagPrefix(cell.Tags, "role-")
		hasEnv := hasTagPrefix(cell.Tags, "env-")
		hasRisk := hasTagPrefix(cell.Tags, "risk-")
		if !hasRole || !hasEnv || !hasRisk {
			missing := []string{}
			if !hasRole {
				missing = append(missing, "role-*")
			}
			if !hasEnv {
				missing = append(missing, "env-*")
			}
			if !hasRisk {
				missing = append(missing, "risk-*")
			}
			report.add("error", "runme_cell_missing_tag_category", cell.Path, "a la celda le faltan categorías de etiqueta obligatorias", map[string]any{"name": cell.Name, "missing": missing, "line": cell.Line})
		}

		mutating := hasTag(cell.Tags, "risk-mutating") || hasTag(cell.Tags, "risk-destructive")
		ciSafe := hasTag(cell.Tags, "ci-safe")
		manualOnly := hasTag(cell.Tags, "manual-only")

		if mutating {
			interactive, hasInteractive := attrBool(cell.Attrs, "interactive")
			exclude, hasExclude := attrBool(cell.Attrs, "excludeFromRunAll")
			if !hasInteractive || !interactive || !hasExclude || !exclude {
				report.add("error", "runme_risk_cell_not_isolated", cell.Path, "una celda risk-mutating/risk-destructive debe ser interactive:true y excludeFromRunAll:true", map[string]any{"name": cell.Name, "line": cell.Line})
			}
			if !manualOnly {
				report.add("error", "runme_risk_cell_not_manual_only", cell.Path, "una celda risk-mutating/risk-destructive debe llevar la etiqueta manual-only", map[string]any{"name": cell.Name, "line": cell.Line})
			}
		}
		if ciSafe && mutating {
			report.add("error", "runme_ci_safe_risk_conflict", cell.Path, "una celda no puede ser ci-safe y risk-mutating/risk-destructive a la vez", map[string]any{"name": cell.Name, "line": cell.Line})
		}
		if ciSafe && manualOnly {
			report.add("error", "runme_contradictory_tags", cell.Path, "una celda no puede ser ci-safe y manual-only a la vez", map[string]any{"name": cell.Name, "line": cell.Line})
		}

		for key := range cell.Attrs {
			if !knownRunmeAttrs[key] {
				report.add("warning", runmeUnknownAttrWarning, cell.Path, "atributo de celda Runme no reconocido por esta validación", map[string]any{"name": cell.Name, "attr": key, "line": cell.Line})
			}
		}

		for _, m := range makeInvocation.FindAllStringSubmatch(cell.Body, -1) {
			if !targets[m[1]] {
				report.add("error", "runme_cell_unknown_make_target", cell.Path, "la celda invoca un target de make que no existe", map[string]any{"name": cell.Name, "target": m[1]})
			}
		}
		for _, m := range scriptInvocation.FindAllStringSubmatch(cell.Body, -1) {
			scriptPath := filepath.Join(root, filepath.FromSlash(m[1]))
			if _, statErr := os.Stat(scriptPath); statErr != nil {
				report.add("error", "runme_cell_unknown_script", cell.Path, "la celda invoca un script que no existe en el repositorio", map[string]any{"name": cell.Name, "script": m[1]})
			}
		}
		if infraToolPattern.MatchString(cell.Body) {
			report.add("warning", "runme_cell_infra_not_present", cell.Path, "la celda referencia una herramienta de infraestructura (kubectl/terraform/helm) que no tiene configuración real en este repositorio", map[string]any{"name": cell.Name})
		}
	}

	a.checkNoRunmeCellsInZolaContent(root, report)
	a.checkRunmeSessionArtifacts(root, report)
}

func mustReadFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// checkNoRunmeCellsInZolaContent hace cumplir la decisión de arquitectura:
// las celdas de Runme viven fuera de content/, nunca dentro. Si alguna
// página publicada por Zola trae atributos de celda Runme, es un error.
func (a *App) checkNoRunmeCellsInZolaContent(root string, report *ValidationReport) {
	sections := []string{"documentacion", "manuales", "memorias", "proyectos", "tareas"}
	for _, section := range sections {
		files, err := listMarkdownFiles(root, section)
		if err != nil {
			continue
		}
		for _, path := range files {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				continue
			}
			for i, line := range strings.Split(string(data), "\n") {
				if m := runmeFenceLine.FindStringSubmatch(strings.TrimRight(line, " \t")); m != nil && m[2] != "" {
					var attrs map[string]any
					if json.Unmarshal([]byte(m[2]), &attrs) == nil {
						if _, ok := attrs["name"]; ok {
							report.add("error", "runme_cell_in_zola_content", path, "content/ es publicado por Zola; las celdas de Runme deben vivir en runbooks/", map[string]any{"line": i + 1})
						}
					}
				}
			}
		}
	}
}

// checkRunmeSessionArtifacts es una defensa best-effort: la documentación
// pública de Runme (docs.runme.dev/usage/auto-save) no fija un nombre de
// archivo exacto para los Session Outputs en v3.17.4, así que en vez de un
// glob amplio en .gitignore (que podría ocultar Markdown legítimo), se
// buscan señales concretas y se documenta la limitación en docs/RUNME.md.
func (a *App) checkRunmeSessionArtifacts(root string, report *ValidationReport) {
	runmeStateDir := filepath.Join(root, ".runme")
	if info, err := os.Stat(runmeStateDir); err == nil && info.IsDir() {
		report.add("error", "runme_state_dir_tracked", runmeStateDir, "existe un directorio .runme en el árbol de trabajo; nunca debe versionarse (estado local de Runme)", nil)
	}
	_ = filepath.WalkDir(filepath.Join(root, "runbooks"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		base := strings.ToLower(filepath.Base(path))
		if strings.Contains(base, "output") && base != "_template.md" {
			report.add("warning", "possible_runme_session_output", path, "el nombre sugiere un Session Output de Runme capturado; confirme que es contenido fuente y no una salida de ejecución antes de conservarlo", nil)
		}
		return nil
	})
}
