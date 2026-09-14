package clusterlog

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newBootstrapApp(t *testing.T, dryRun bool) *App {
	t.Helper()
	return New(Options{DryRun: dryRun, JSON: true, Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin, Executable: "clusterlog"})
}

func TestBootstrapInstanceCreatesExpectedFiles(t *testing.T) {
	base := t.TempDir()
	out := filepath.Join(base, "instancia-nueva")
	app := newBootstrapApp(t, false)

	err := app.runBootstrapInstance([]string{"--output", out, "--product-name", "Bitácora de Prueba", "--base-url", "https://clusterlog.prueba.org"})
	if err != nil {
		t.Fatalf("runBootstrapInstance falló: %v", err)
	}

	for _, section := range scaffoldSections {
		path := filepath.Join(out, "content", section, "_index.md")
		if _, statErr := os.Stat(path); statErr != nil {
			t.Fatalf("falta %s: %v", path, statErr)
		}
	}

	homeData, err := os.ReadFile(filepath.Join(out, "content", "_index.md"))
	if err != nil {
		t.Fatalf("falta la portada content/_index.md (necesaria para que zola build no falle): %v", err)
	}
	if !strings.Contains(string(homeData), `title = "Bitácora de Prueba"`) {
		t.Fatalf("la portada no sustituyó TITLE: %s", homeData)
	}

	zolaData, err := os.ReadFile(filepath.Join(out, "zola.toml"))
	if err != nil {
		t.Fatal(err)
	}
	zola := string(zolaData)
	if !strings.Contains(zola, `base_url = "https://clusterlog.prueba.org"`) {
		t.Fatalf("zola.toml no tiene el base_url esperado: %s", zola)
	}
	if !strings.Contains(zola, `title = "Bitácora de Prueba"`) {
		t.Fatalf("zola.toml no tiene el title esperado: %s", zola)
	}
	if !strings.Contains(zola, `product_name = "Bitácora de Prueba"`) {
		t.Fatalf("zola.toml no tiene product_name esperado: %s", zola)
	}

	var registry AdminRegistry
	adminsData, err := os.ReadFile(filepath.Join(out, "data", "admins.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(adminsData, &registry); err != nil {
		t.Fatal(err)
	}
	if registry.Version != 1 || len(registry.Admins) != 0 || registry.ReviewPolicy.Default != "all-active" {
		t.Fatalf("registro de admins inesperado: %+v", registry)
	}

	if _, err := os.Stat(filepath.Join(out, "config", "allowed_signers")); err != nil {
		t.Fatalf("falta config/allowed_signers: %v", err)
	}

	versionData, err := os.ReadFile(filepath.Join(out, ".clusterlog-version"))
	if err != nil {
		t.Fatal(err)
	}
	if trimmed := strings.TrimSpace(string(versionData)); trimmed != Version {
		t.Fatalf(".clusterlog-version = %q, se esperaba %q", trimmed, Version)
	}

	for _, dir := range []string{"review-events", "task-events"} {
		if _, err := os.Stat(filepath.Join(out, "data", dir, ".gitkeep")); err != nil {
			t.Fatalf("falta data/%s/.gitkeep: %v", dir, err)
		}
	}

	if _, err := os.Stat(filepath.Join(out, "README.md")); err != nil {
		t.Fatalf("falta README.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, ".gitignore")); err != nil {
		t.Fatalf("falta .gitignore: %v", err)
	}
}

func TestBootstrapInstanceScaffoldValidatesClean(t *testing.T) {
	base := t.TempDir()
	out := filepath.Join(base, "instancia-nueva")
	app := newBootstrapApp(t, false)
	if err := app.runBootstrapInstance([]string{"--output", out}); err != nil {
		t.Fatalf("runBootstrapInstance falló: %v", err)
	}

	report := ValidationReport{Checks: map[string]int{}, Errors: []ValidationIssue{}, Warnings: []ValidationIssue{}}
	validateApp := &App{}
	registry, err := loadAdmins(out)
	if err != nil {
		t.Fatal(err)
	}
	validateApp.validateAdmins(out, registry, &report)
	pages := validateApp.validatePages(out, registry, &report)
	validateApp.validateEvents(out, registry, pages, &report)
	if len(report.Errors) != 0 {
		t.Fatalf("el scaffold recién creado no debería tener errores de validate: %+v", report.Errors)
	}
}

func TestBootstrapInstanceRefusesNonEmptyOutput(t *testing.T) {
	base := t.TempDir()
	out := filepath.Join(base, "instancia-existente")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteTestFile(t, filepath.Join(out, "algo.txt"), "ya había algo aquí")

	app := newBootstrapApp(t, false)
	err := app.runBootstrapInstance([]string{"--output", out})
	if err == nil {
		t.Fatal("se esperaba un error por directorio no vacío")
	}
	cliErr, ok := err.(*CLIError)
	if !ok || cliErr.Kind != "output_not_empty" {
		t.Fatalf("se esperaba output_not_empty, se obtuvo: %v", err)
	}
}

func TestBootstrapInstanceDryRunWritesNothing(t *testing.T) {
	base := t.TempDir()
	out := filepath.Join(base, "instancia-dry-run")
	app := newBootstrapApp(t, true)
	if err := app.runBootstrapInstance([]string{"--output", out}); err != nil {
		t.Fatalf("runBootstrapInstance --dry-run falló: %v", err)
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("--dry-run no debería haber creado el directorio de salida")
	}
}

func TestBootstrapInstanceNextStepsDoNotEscapeAngleBrackets(t *testing.T) {
	base := t.TempDir()
	out := filepath.Join(base, "instancia-nueva")
	var stdout bytes.Buffer
	app := New(Options{JSON: true, Stdout: &stdout, Stderr: os.Stderr, Stdin: os.Stdin, Executable: "clusterlog"})

	if err := app.runBootstrapInstance([]string{"--output", out}); err != nil {
		t.Fatalf("runBootstrapInstance falló: %v", err)
	}

	printed := stdout.String()
	escapedLT := string([]byte{'\\', 'u', '0', '0', '3', 'c'})
	escapedGT := string([]byte{'\\', 'u', '0', '0', '3', 'e'})
	if strings.Contains(printed, escapedLT) || strings.Contains(printed, escapedGT) {
		t.Fatalf("la salida JSON escapó < y > en vez de dejarlos literales (encoding/json con SetEscapeHTML por defecto): %s", printed)
	}
	if !strings.Contains(printed, "<su-llave.pub>") {
		t.Fatalf("se esperaba el placeholder <su-llave.pub> literal en next_steps: %s", printed)
	}
}

func TestBootstrapInstanceRequiresOutput(t *testing.T) {
	app := newBootstrapApp(t, false)
	err := app.runBootstrapInstance([]string{})
	if err == nil {
		t.Fatal("se esperaba un error por --output faltante")
	}
	cliErr, ok := err.(*CLIError)
	if !ok || cliErr.Kind != "missing_output" {
		t.Fatalf("se esperaba missing_output, se obtuvo: %v", err)
	}
}
