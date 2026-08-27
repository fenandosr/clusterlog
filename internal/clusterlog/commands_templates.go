package clusterlog

import (
	"path/filepath"
	"sort"
	"strings"
)

// runTemplates despacha los subcomandos de "templates". Por ahora sólo
// existe "extract".
func (a *App) runTemplates(root string, args []string) error {
	if len(args) == 0 {
		return NewError(ExitUsage, "missing_subcommand", "uso: templates extract [--output DIR]", nil)
	}
	switch args[0] {
	case "extract":
		return a.runTemplatesExtract(root, args[1:])
	default:
		return NewError(ExitUsage, "unknown_subcommand", "subcomando desconocido: templates "+args[0], nil)
	}
}

// runTemplatesExtract escribe las plantillas Tera embebidas en el binario
// (internal/clusterlog/scaffold/templates) en el directorio destino,
// normalmente <root>/templates. No requiere red: las plantillas viajan
// dentro del propio binario, en la misma versión exacta que lo compiló, así
// que nunca pueden desincronizarse entre sí.
func (a *App) runTemplatesExtract(root string, args []string) error {
	fs := newFlagSet("templates extract")
	output := fs.String("output", "", "directorio destino (por defecto <root>/templates)")
	if err := fs.Parse(args); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}
	dest := strings.TrimSpace(*output)
	if dest == "" {
		dest = filepath.Join(root, "templates")
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return NewError(ExitInternal, "internal", err.Error(), nil)
	}

	written, err := writeEmbeddedTemplates(absDest)
	if err != nil {
		return err
	}

	return a.printResult("templates.extract", map[string]any{
		"output":         absDest,
		"engine_version": Version,
		"files_written":  written,
	})
}

func writeEmbeddedTemplates(dest string) ([]string, error) {
	src := scaffoldRoot + "/templates"
	entries, err := scaffoldFS.ReadDir(src)
	if err != nil {
		return nil, NewError(ExitInternal, "internal", "plantillas embebidas faltantes", err.Error())
	}
	written := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, readErr := scaffoldFS.ReadFile(src + "/" + entry.Name())
		if readErr != nil {
			return nil, NewError(ExitInternal, "internal", "no se pudo leer una plantilla embebida", map[string]any{"name": entry.Name(), "error": readErr.Error()})
		}
		path := filepath.Join(dest, entry.Name())
		if err := writeFileAtomic(path, data, 0o644); err != nil {
			return nil, NewError(ExitInternal, "internal", err.Error(), path)
		}
		written = append(written, path)
	}
	sort.Strings(written)
	return written, nil
}
