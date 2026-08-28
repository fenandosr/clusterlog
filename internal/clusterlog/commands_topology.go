package clusterlog

import (
	"path/filepath"
	"strings"
)

// runTopology despacha los subcomandos de "topology". Por ahora sólo existe
// "import", y sólo con el adaptador --from-mksrv (ver topology.go).
func (a *App) runTopology(root string, args []string) error {
	if len(args) == 0 {
		return NewError(ExitUsage, "missing_subcommand", "uso: topology import --from-mksrv DIR [--output PATH] [--include-public-ip]", nil)
	}
	switch args[0] {
	case "import":
		return a.runTopologyImport(root, args[1:])
	default:
		return NewError(ExitUsage, "unknown_subcommand", "subcomando desconocido: topology "+args[0], nil)
	}
}

// runTopologyImport lee un workspace mksrv y escribe data/topology.json.
// No firma ni comitea nada (igual que "templates extract"): es un paso de
// generación, no de contenido auditado.
func (a *App) runTopologyImport(root string, args []string) error {
	fs := newFlagSet("topology import")
	fromMksrv := fs.String("from-mksrv", "", "ruta al workspace de mksrv (con deployment.yaml y .mksrv/infra/outputs.json)")
	output := fs.String("output", "", "archivo destino (por defecto <root>/data/topology.json)")
	includePublicIP := fs.Bool("include-public-ip", false, "incluir la IP pública de cada host (decisión de contenido explícita, no automática)")
	if err := fs.Parse(args); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}

	src := strings.TrimSpace(*fromMksrv)
	if src == "" {
		return NewError(ExitUsage, "missing_source", "topology import requiere --from-mksrv DIR; es el único adaptador soportado hoy", nil)
	}

	topo, warnings, err := importFromMksrv(src, *includePublicIP)
	if err != nil {
		return err
	}

	dest := strings.TrimSpace(*output)
	if dest == "" {
		dest = filepath.Join(root, "data", "topology.json")
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return NewError(ExitInternal, "internal", err.Error(), nil)
	}

	if a.Options.DryRun {
		return a.printResult("topology.import", map[string]any{
			"output":   absDest,
			"source":   topo.Source,
			"hosts":    len(topo.Hosts),
			"topology": topo,
			"dry_run":  true,
		}, warnings...)
	}

	if err := writeJSONAtomic(absDest, topo); err != nil {
		return NewError(ExitInternal, "internal", err.Error(), absDest)
	}

	return a.printResult("topology.import", map[string]any{
		"output": absDest,
		"source": topo.Source,
		"hosts":  len(topo.Hosts),
	}, warnings...)
}
