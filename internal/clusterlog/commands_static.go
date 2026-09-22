package clusterlog

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// runStatic despacha los subcomandos de "static". Por ahora sólo existe
// "extract".
func (a *App) runStatic(root string, args []string) error {
	if len(args) == 0 {
		return NewError(ExitUsage, "missing_subcommand", "uso: static extract [--output DIR]", nil)
	}
	switch args[0] {
	case "extract":
		return a.runStaticExtract(root, args[1:])
	default:
		return NewError(ExitUsage, "unknown_subcommand", "subcomando desconocido: static "+args[0], nil)
	}
}

// runStaticExtract escribe los assets estáticos embebidos en el binario
// (internal/clusterlog/scaffold/static -- ver ADR-0009) en el directorio
// destino, normalmente <root>/static. No requiere red: los assets viajan
// dentro del propio binario, en la misma versión exacta que lo compiló.
func (a *App) runStaticExtract(root string, args []string) error {
	fs := newFlagSet("static extract")
	output := fs.String("output", "", "directorio destino (por defecto <root>/static)")
	if err := fs.Parse(args); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}
	dest := strings.TrimSpace(*output)
	if dest == "" {
		dest = filepath.Join(root, "static")
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return NewError(ExitInternal, "internal", err.Error(), nil)
	}

	written, err := writeEmbeddedStatic(absDest)
	if err != nil {
		return err
	}

	return a.printResult("static.extract", map[string]any{
		"output":         absDest,
		"engine_version": Version,
		"files_written":  written,
	})
}

// writeEmbeddedStatic copia recursivamente internal/clusterlog/scaffold/static
// (css/, js/, fonts/, robots.txt) a dest. A diferencia de
// writeEmbeddedTemplates, sí recorre subdirectorios: css/js/fonts son
// parte real del contrato de estos assets, no un detalle interno.
func writeEmbeddedStatic(dest string) ([]string, error) {
	src := scaffoldRoot + "/static"
	written := []string{}
	err := fs.WalkDir(scaffoldFS, src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, readErr := scaffoldFS.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel := strings.TrimPrefix(path, src+"/")
		out := filepath.Join(dest, filepath.FromSlash(rel))
		if err := writeFileAtomic(out, data, 0o644); err != nil {
			return err
		}
		written = append(written, out)
		return nil
	})
	if err != nil {
		return nil, NewError(ExitInternal, "internal", "no se pudieron extraer los assets estáticos embebidos", err.Error())
	}
	sort.Strings(written)
	return written, nil
}
