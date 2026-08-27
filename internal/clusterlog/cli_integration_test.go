package clusterlog

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runJSONCommand(t *testing.T, root, keyPath, stdin string, now time.Time, args ...string) ResultEnvelope {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := New(Options{
		Root: root, JSON: true, SSHKey: keyPath, Stdout: &stdout, Stderr: &stderr,
		Stdin: strings.NewReader(stdin), Now: func() time.Time { return now }, Executable: "clusterlog",
	})
	if err := app.Run(args); err != nil {
		t.Fatalf("command %v failed: %v\nstderr: %s", args, err, stderr.String())
	}
	var envelope ResultEnvelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("invalid command JSON for %v: %v\n%s", args, err, stdout.String())
	}
	if !envelope.OK {
		t.Fatalf("command returned non-ok envelope: %#v", envelope)
	}
	return envelope
}

func envelopeDataMap(t *testing.T, envelope ResultEnvelope) map[string]any {
	t.Helper()
	data, ok := envelope.Data.(map[string]any)
	if !ok {
		t.Fatalf("unexpected data type: %T", envelope.Data)
	}
	return data
}

func TestCLIAdminMemoryReviewFlow(t *testing.T) {
	root := testProjectRoot(t)
	keyPath := filepath.Join(root, "ana.pub")
	if err := os.WriteFile(keyPath, []byte(testOpenSSHPublicKey(9, "ana@test")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 12, 14, 0, 0, 123, time.UTC)
	runJSONCommand(t, root, keyPath, "", now,
		"admin", "add", "--id", "ana", "--name", "Ana Operadora", "--email", "ana@example.test")

	memoryJSON := `{
	  "title": "Reiniciar controlador de almacenamiento",
	  "description": "Corrección controlada de un controlador degradado",
	  "risk": "high",
	  "tags": ["storage", "mantenimiento"],
	  "systems": ["ceph"],
	  "changes": ["Se drenó el nodo", "Se reinició el controlador"],
	  "validation": ["El clúster volvió a HEALTH_OK"],
	  "rollback": "Reintegrar el controlador anterior",
	  "agent": "claude-code"
	}`
	envelope := runJSONCommand(t, root, keyPath, memoryJSON, now.Add(time.Minute),
		"memory", "create", "--from-json", "-")
	data := envelopeDataMap(t, envelope)
	entryID, ok := data["entry_id"].(string)
	if !ok || entryID == "" {
		t.Fatalf("missing entry id: %#v", data)
	}
	state, err := New(Options{}).BuildState(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.ReviewQueue) != 1 || !contains(state.ReviewQueue[0].MissingReviewers, "ana") {
		t.Fatalf("unexpected review queue: %#v", state.ReviewQueue)
	}

	runJSONCommand(t, root, keyPath, "", now.Add(2*time.Minute), "review", "mark", entryID, "--note", "Validado")
	state, err = New(Options{}).BuildState(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.ReviewQueue) != 0 {
		t.Fatalf("reviewed item remained pending: %#v", state.ReviewQueue)
	}
	files, err := reviewEventFiles(root)
	if err != nil || len(files) != 1 {
		t.Fatalf("expected one independent review event file, got %v, err=%v", files, err)
	}
}
