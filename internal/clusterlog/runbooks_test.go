package clusterlog

import (
	"os"
	"path/filepath"
	"testing"
)

func newRunbookTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWriteTestFile(t, filepath.Join(root, "Makefile"), ".PHONY: build test\nbuild:\n\techo build\ntest:\n\techo test\n")
	mustWriteTestFile(t, filepath.Join(root, "scripts", "smoke.sh"), "#!/usr/bin/env bash\necho ok\n")
	return root
}

func runRunbookValidation(t *testing.T, root string) *ValidationReport {
	t.Helper()
	report := &ValidationReport{Checks: map[string]int{}, Errors: []ValidationIssue{}, Warnings: []ValidationIssue{}}
	app := &App{}
	app.validateRunbooks(root, report)
	return report
}

func hasErrorCode(report *ValidationReport, code string) bool {
	for _, issue := range report.Errors {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func TestValidateRunbooksAcceptsWellFormedCell(t *testing.T) {
	root := newRunbookTestRoot(t)
	mustWriteTestFile(t, filepath.Join(root, "RUNBOOK.md"), "# Runbook\n\n"+
		"```sh {\"name\":\"clusterlog-build\",\"tag\":\"role-developer,env-local,risk-read-only,ci-safe\",\"interactive\":\"false\"}\n"+
		"make build\n"+
		"```\n")
	report := runRunbookValidation(t, root)
	if len(report.Errors) != 0 {
		t.Fatalf("no se esperaban errores, hay: %+v", report.Errors)
	}
	if report.Checks["runme_cells"] != 1 {
		t.Fatalf("se esperaba contar 1 celda, hay %d", report.Checks["runme_cells"])
	}
}

func TestValidateRunbooksDetectsDuplicateNames(t *testing.T) {
	root := newRunbookTestRoot(t)
	cell := "```sh {\"name\":\"clusterlog-build\",\"tag\":\"role-developer,env-local,risk-read-only,ci-safe\",\"interactive\":\"false\"}\nmake build\n```\n"
	mustWriteTestFile(t, filepath.Join(root, "RUNBOOK.md"), "# Runbook\n\n"+cell)
	mustWriteTestFile(t, filepath.Join(root, "runbooks", "clusterlog-development.md"), "# Dev\n\n"+cell)
	report := runRunbookValidation(t, root)
	if !hasErrorCode(report, "duplicate_runme_cell_name") {
		t.Fatalf("se esperaba duplicate_runme_cell_name, errores: %+v", report.Errors)
	}
}

func TestValidateRunbooksRequiresTagCategories(t *testing.T) {
	root := newRunbookTestRoot(t)
	mustWriteTestFile(t, filepath.Join(root, "RUNBOOK.md"), "# Runbook\n\n"+
		"```sh {\"name\":\"clusterlog-build\",\"tag\":\"ci-safe\",\"interactive\":\"false\"}\n"+
		"make build\n"+
		"```\n")
	report := runRunbookValidation(t, root)
	if !hasErrorCode(report, "runme_cell_missing_tag_category") {
		t.Fatalf("se esperaba runme_cell_missing_tag_category, errores: %+v", report.Errors)
	}
}

func TestValidateRunbooksRequiresIsolationForMutatingCells(t *testing.T) {
	root := newRunbookTestRoot(t)
	mustWriteTestFile(t, filepath.Join(root, "RUNBOOK.md"), "# Runbook\n\n"+
		"```sh {\"name\":\"clusterlog-memory-create\",\"tag\":\"role-operator,env-production,risk-mutating,manual-only\"}\n"+
		"make build\n"+
		"```\n")
	report := runRunbookValidation(t, root)
	if !hasErrorCode(report, "runme_risk_cell_not_isolated") {
		t.Fatalf("se esperaba runme_risk_cell_not_isolated, errores: %+v", report.Errors)
	}
}

func TestValidateRunbooksRejectsCiSafeMutatingConflict(t *testing.T) {
	root := newRunbookTestRoot(t)
	mustWriteTestFile(t, filepath.Join(root, "RUNBOOK.md"), "# Runbook\n\n"+
		"```sh {\"name\":\"clusterlog-memory-create\",\"tag\":\"role-operator,env-production,risk-mutating,ci-safe\",\"interactive\":\"true\",\"excludeFromRunAll\":true}\n"+
		"make build\n"+
		"```\n")
	report := runRunbookValidation(t, root)
	if !hasErrorCode(report, "runme_ci_safe_risk_conflict") {
		t.Fatalf("se esperaba runme_ci_safe_risk_conflict, errores: %+v", report.Errors)
	}
	if !hasErrorCode(report, "runme_risk_cell_not_manual_only") {
		t.Fatalf("se esperaba runme_risk_cell_not_manual_only, errores: %+v", report.Errors)
	}
}

func TestValidateRunbooksRejectsUnknownMakeTarget(t *testing.T) {
	root := newRunbookTestRoot(t)
	mustWriteTestFile(t, filepath.Join(root, "RUNBOOK.md"), "# Runbook\n\n"+
		"```sh {\"name\":\"clusterlog-deploy\",\"tag\":\"role-developer,env-local,risk-read-only,ci-safe\",\"interactive\":\"false\"}\n"+
		"make deploy-to-mars\n"+
		"```\n")
	report := runRunbookValidation(t, root)
	if !hasErrorCode(report, "runme_cell_unknown_make_target") {
		t.Fatalf("se esperaba runme_cell_unknown_make_target, errores: %+v", report.Errors)
	}
}

func TestValidateRunbooksRejectsUnknownScript(t *testing.T) {
	root := newRunbookTestRoot(t)
	mustWriteTestFile(t, filepath.Join(root, "RUNBOOK.md"), "# Runbook\n\n"+
		"```sh {\"name\":\"clusterlog-preflight\",\"tag\":\"role-developer,env-local,risk-read-only,ci-safe\",\"interactive\":\"false\"}\n"+
		"./scripts/does-not-exist.sh\n"+
		"```\n")
	report := runRunbookValidation(t, root)
	if !hasErrorCode(report, "runme_cell_unknown_script") {
		t.Fatalf("se esperaba runme_cell_unknown_script, errores: %+v", report.Errors)
	}
}

func TestValidateRunbooksDetectsSecrets(t *testing.T) {
	root := newRunbookTestRoot(t)
	mustWriteTestFile(t, filepath.Join(root, "RUNBOOK.md"), "# Runbook\n\n"+
		"```sh {\"name\":\"clusterlog-leak\",\"tag\":\"role-developer,env-local,risk-read-only,ci-safe\",\"interactive\":\"false\"}\n"+
		"export password: hunter2-super-secret\n"+
		"```\n")
	report := runRunbookValidation(t, root)
	if !hasErrorCode(report, "possible_secret") {
		t.Fatalf("se esperaba possible_secret, errores: %+v", report.Errors)
	}
}

func TestValidateRunbooksRejectsRunmeCellsInsideZolaContent(t *testing.T) {
	root := newRunbookTestRoot(t)
	if err := os.MkdirAll(filepath.Join(root, "content", "manuales"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteTestFile(t, filepath.Join(root, "content", "manuales", "ejemplo.md"), "+++\ntitle = \"x\"\n+++\n\n"+
		"```sh {\"name\":\"clusterlog-build\",\"tag\":\"role-developer,env-local,risk-read-only,ci-safe\"}\n"+
		"make build\n"+
		"```\n")
	report := runRunbookValidation(t, root)
	if !hasErrorCode(report, "runme_cell_in_zola_content") {
		t.Fatalf("se esperaba runme_cell_in_zola_content, errores: %+v", report.Errors)
	}
}

func TestValidateRunbooksRejectsTrackedRunmeStateDir(t *testing.T) {
	root := newRunbookTestRoot(t)
	mustWriteTestFile(t, filepath.Join(root, "RUNBOOK.md"), "# Runbook\n")
	if err := os.MkdirAll(filepath.Join(root, ".runme"), 0o755); err != nil {
		t.Fatal(err)
	}
	report := runRunbookValidation(t, root)
	if !hasErrorCode(report, "runme_state_dir_tracked") {
		t.Fatalf("se esperaba runme_state_dir_tracked, errores: %+v", report.Errors)
	}
}

func TestValidateRunbooksNoRunbooksIsNoop(t *testing.T) {
	root := t.TempDir()
	report := runRunbookValidation(t, root)
	if len(report.Errors) != 0 || len(report.Warnings) != 0 {
		t.Fatalf("sin runbooks no debería haber hallazgos: %+v / %+v", report.Errors, report.Warnings)
	}
}
