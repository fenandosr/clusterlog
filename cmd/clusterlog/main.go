package main

import (
	"flag"
	"fmt"
	"os"

	"example.org/bitacora-cluster/internal/clusterlog"
)

func main() {
	fs := flag.NewFlagSet("clusterlog", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", "", "raíz del proyecto")
	jsonMode := fs.Bool("json", false, "salida JSON estable")
	sshKey := fs.String("ssh-key", "", "llave pública SSH o ruta base")
	commit := fs.Bool("commit", false, "crea un commit Git SSH firmado")
	dryRun := fs.Bool("dry-run", false, "valida sin escribir")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Uso: clusterlog [opciones globales] <comando> [opciones]")
		fmt.Fprintln(os.Stderr, "Ejecute clusterlog help para ver los comandos.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(clusterlog.PrintError(os.Stderr, *jsonMode, clusterlog.NewError(clusterlog.ExitUsage, "invalid_global_flags", err.Error(), nil)))
	}
	app := clusterlog.New(clusterlog.Options{
		Root: *root, JSON: *jsonMode, SSHKey: *sshKey, Commit: *commit, DryRun: *dryRun,
		Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin, Executable: "clusterlog",
	})
	if err := app.Run(fs.Args()); err != nil {
		os.Exit(clusterlog.PrintError(os.Stderr, *jsonMode, err))
	}
}
