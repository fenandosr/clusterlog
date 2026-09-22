package clusterlog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStaticExtractWritesAllFiles(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "instancia")
	mustWriteTestFile(t, filepath.Join(root, "zola.toml"), "title = \"x\"\n")

	app := New(Options{JSON: true, Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin, Executable: "clusterlog"})
	if err := app.runStatic(root, []string{"extract"}); err != nil {
		t.Fatalf("static extract falló: %v", err)
	}

	for _, want := range []string{
		filepath.Join("css", "ops.css"),
		filepath.Join("js", "ops-search.js"),
		filepath.Join("fonts", "ShareTechMono-Regular.woff2"),
		"robots.txt",
	} {
		if _, err := os.Stat(filepath.Join(root, "static", want)); err != nil {
			t.Fatalf("falta %s: %v", want, err)
		}
	}
}

func TestStaticExtractRespectsOutputFlag(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "instancia")
	out := filepath.Join(base, "otro-destino")
	mustWriteTestFile(t, filepath.Join(root, "zola.toml"), "title = \"x\"\n")

	app := New(Options{JSON: true, Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin, Executable: "clusterlog"})
	if err := app.runStatic(root, []string{"extract", "--output", out}); err != nil {
		t.Fatalf("static extract --output falló: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "css", "ops.css")); err != nil {
		t.Fatalf("no escribió en --output: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "static")); err == nil {
		t.Fatal("no debería haber escrito también en <root>/static")
	}
}

func TestStaticUnknownSubcommand(t *testing.T) {
	app := New(Options{JSON: true, Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin, Executable: "clusterlog"})
	err := app.runStatic(t.TempDir(), []string{"bogus"})
	if err == nil {
		t.Fatal("se esperaba un error por subcomando desconocido")
	}
	cliErr, ok := err.(*CLIError)
	if !ok || cliErr.Kind != "unknown_subcommand" {
		t.Fatalf("se esperaba unknown_subcommand, se obtuvo: %v", err)
	}
}

func TestBootstrapInstanceIncludesStaticAssets(t *testing.T) {
	base := t.TempDir()
	out := filepath.Join(base, "instancia-nueva")
	app := newBootstrapApp(t, false)
	if err := app.runBootstrapInstance([]string{"--output", out}); err != nil {
		t.Fatalf("runBootstrapInstance falló: %v", err)
	}
	for _, want := range []string{
		filepath.Join("css", "ops.css"),
		filepath.Join("js", "ops-search.js"),
		filepath.Join("fonts", "ShareTechMono-Regular.woff2"),
	} {
		if _, err := os.Stat(filepath.Join(out, "static", want)); err != nil {
			t.Fatalf("bootstrap-instance no incluyó static/%s: %v", want, err)
		}
	}
}
