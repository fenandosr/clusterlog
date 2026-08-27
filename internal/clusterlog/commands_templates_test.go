package clusterlog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTemplatesExtractWritesAllFiles(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "instancia")
	mustWriteTestFile(t, filepath.Join(root, "zola.toml"), "title = \"x\"\n")

	app := New(Options{JSON: true, Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin, Executable: "clusterlog"})
	if err := app.runTemplates(root, []string{"extract"}); err != nil {
		t.Fatalf("templates extract falló: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(root, "templates"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no se escribió ninguna plantilla")
	}
	for _, want := range []string{"base.html", "index.html", "memory.html", "page.html"} {
		if _, err := os.Stat(filepath.Join(root, "templates", want)); err != nil {
			t.Fatalf("falta %s: %v", want, err)
		}
	}
}

func TestTemplatesExtractRespectsOutputFlag(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "instancia")
	out := filepath.Join(base, "otro-destino")
	mustWriteTestFile(t, filepath.Join(root, "zola.toml"), "title = \"x\"\n")

	app := New(Options{JSON: true, Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin, Executable: "clusterlog"})
	if err := app.runTemplates(root, []string{"extract", "--output", out}); err != nil {
		t.Fatalf("templates extract --output falló: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "base.html")); err != nil {
		t.Fatalf("no escribió en --output: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "templates")); err == nil {
		t.Fatal("no debería haber escrito también en <root>/templates")
	}
}

func TestTemplatesUnknownSubcommand(t *testing.T) {
	app := New(Options{JSON: true, Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin, Executable: "clusterlog"})
	err := app.runTemplates(t.TempDir(), []string{"bogus"})
	if err == nil {
		t.Fatal("se esperaba un error por subcomando desconocido")
	}
	cliErr, ok := err.(*CLIError)
	if !ok || cliErr.Kind != "unknown_subcommand" {
		t.Fatalf("se esperaba unknown_subcommand, se obtuvo: %v", err)
	}
}

func TestBootstrapInstanceIncludesTemplates(t *testing.T) {
	base := t.TempDir()
	out := filepath.Join(base, "instancia-nueva")
	app := newBootstrapApp(t, false)
	if err := app.runBootstrapInstance([]string{"--output", out}); err != nil {
		t.Fatalf("runBootstrapInstance falló: %v", err)
	}
	for _, want := range []string{"base.html", "index.html", "memory.html"} {
		if _, err := os.Stat(filepath.Join(out, "templates", want)); err != nil {
			t.Fatalf("bootstrap-instance no incluyó %s: %v", want, err)
		}
	}
}
