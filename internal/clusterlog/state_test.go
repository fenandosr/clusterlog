package clusterlog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReviewQueueUsesContentHashAndReviewerSnapshot(t *testing.T) {
	root := testProjectRoot(t)
	ana := testAdmin(t, "ana", 1, "owner", "admin")
	bob := testAdmin(t, "bob", 2, "admin")
	registry := AdminRegistry{
		Version:      1,
		ReviewPolicy: ReviewPolicy{Default: "all-active", IncludeAuthor: true},
		Admins:       []Admin{ana, bob},
	}
	if err := saveAdmins(root, registry); err != nil {
		t.Fatal(err)
	}
	memoryPath := filepath.Join(root, "content", "memorias", "memory.md")
	mustWriteTestFile(t, memoryPath, `+++
title = "Cambio de red"
description = "Ajuste controlado"
date = 2026-08-12T10:00:00Z
authors = ["ana"]
slug = "cambio-red"

[taxonomies]
tags = ["red"]
systems = ["spine-01"]

[extra]
id = "MEM-TEST-001"
author_id = "ana"
author_fingerprint = "`+ana.SSHKeys[0].Fingerprint+`"
risk = "high"
review_required = true
review_policy = "all-active"
reviewer_snapshot = true
reviewers = ["ana", "bob"]
+++

## Cambio

Versión uno.
`)
	app := New(Options{Root: root, Now: func() time.Time { return time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC) }})
	state, err := app.BuildState(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.ReviewQueue) != 1 || len(state.ReviewQueue[0].MissingReviewers) != 2 {
		t.Fatalf("unexpected initial queue: %#v", state.ReviewQueue)
	}
	memory, _, err := findMemory(root, "MEM-TEST-001")
	if err != nil {
		t.Fatal(err)
	}
	event := ReviewEvent{
		ID: "REV-TEST-001", EntryID: memory.EntryID, AdminID: "ana",
		Fingerprint: ana.SSHKeys[0].Fingerprint, ReviewedAt: "2026-08-12T12:01:00Z",
		ContentSHA256: memory.ContentSHA256,
	}
	if _, err := writeReviewEvent(root, event); err != nil {
		t.Fatal(err)
	}
	state, err = app.BuildState(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := state.ReviewQueue[0].MissingReviewers; len(got) != 1 || got[0] != "bob" {
		t.Fatalf("unexpected missing reviewers after review: %#v", got)
	}

	// Adding a new administrator must not reopen old entries because the
	// required reviewer set is captured when the memory is created.
	carol := testAdmin(t, "carol", 3, "admin")
	registry.Admins = append(registry.Admins, carol)
	if err := saveAdmins(root, registry); err != nil {
		t.Fatal(err)
	}
	state, err = app.BuildState(root)
	if err != nil {
		t.Fatal(err)
	}
	if state.ReviewQueue[0].RequiredCount != 2 || contains(state.ReviewQueue[0].MissingReviewers, "carol") {
		t.Fatalf("snapshot unexpectedly changed: %#v", state.ReviewQueue[0])
	}

	// Any edit changes the hash, invalidating the old review automatically.
	data, err := os.ReadFile(memoryPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(memoryPath, append(data, []byte("\nCorrección posterior.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	state, err = app.BuildState(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := state.ReviewQueue[0].MissingReviewers; len(got) != 2 || !contains(got, "ana") || !contains(got, "bob") {
		t.Fatalf("old reviews were not invalidated: %#v", got)
	}
}

func TestTaskProjectionFromIndependentEventFiles(t *testing.T) {
	root := testProjectRoot(t)
	ana := testAdmin(t, "ana", 1, "owner", "admin")
	if err := saveAdmins(root, AdminRegistry{Version: 1, ReviewPolicy: ReviewPolicy{Default: "all-active", IncludeAuthor: true}, Admins: []Admin{ana}}); err != nil {
		t.Fatal(err)
	}
	mustWriteTestFile(t, filepath.Join(root, "content", "tareas", "task.md"), `+++
title = "Probar respaldo"
description = "Prueba de tarea"
date = 2026-08-12T10:00:00Z
authors = ["ana"]
slug = "probar-respaldo"

[taxonomies]
tags = ["backup"]
systems = ["etcd"]

[extra]
id = "TSK-TEST-001"
status = "todo"
priority = "high"
project_id = ""
assignees = ["ana"]
estimate_minutes = 60
due = "2026-08-20"
+++

## Objetivo

Validar el respaldo.
`)
	events := []TaskEvent{
		{ID: "EVT-3", TaskID: "TSK-TEST-001", Type: "status", AdminID: "ana", Fingerprint: ana.SSHKeys[0].Fingerprint, At: "2026-08-12T10:30:00Z", Status: "blocked"},
		{ID: "EVT-1", TaskID: "TSK-TEST-001", Type: "start", AdminID: "ana", Fingerprint: ana.SSHKeys[0].Fingerprint, At: "2026-08-12T10:00:00Z"},
		{ID: "EVT-2", TaskID: "TSK-TEST-001", Type: "stop", AdminID: "ana", Fingerprint: ana.SSHKeys[0].Fingerprint, At: "2026-08-12T10:25:00Z", Minutes: 25},
		{ID: "EVT-4", TaskID: "TSK-TEST-001", Type: "adjust", AdminID: "ana", Fingerprint: ana.SSHKeys[0].Fingerprint, At: "2026-08-12T10:31:00Z", Minutes: 5},
	}
	// Intentionally write out of chronological order: loading must sort by At.
	if _, err := writeTaskEvents(root, events); err != nil {
		t.Fatal(err)
	}
	state, err := New(Options{}).BuildState(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Tasks) != 1 {
		t.Fatalf("unexpected tasks: %#v", state.Tasks)
	}
	got := state.Tasks[0]
	if got.ActualMinutes != 30 || got.Status != "blocked" || len(got.RunningBy) != 0 || got.RunningBy == nil {
		t.Fatalf("unexpected task projection: %#v", got)
	}
}

func TestSyncRegeneratesWhenProjectionVersionChanges(t *testing.T) {
	root := testProjectRoot(t)
	app := New(Options{Root: root, Now: func() time.Time { return time.Date(2026, 8, 12, 15, 0, 0, 0, time.UTC) }})
	state, changed, path, err := app.Sync(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || state.Version != siteStateVersion {
		t.Fatalf("unexpected initial sync: changed=%v state=%#v", changed, state)
	}

	var stale SiteState
	if err := loadJSON(path, &stale); err != nil {
		t.Fatal(err)
	}
	stale.Version = siteStateVersion - 1
	if err := writeJSONAtomic(path, stale); err != nil {
		t.Fatal(err)
	}

	if _, _, _, err := app.Sync(root, true); err == nil {
		t.Fatal("sync --check accepted a stale projection version")
	}
	state, changed, _, err = app.Sync(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || state.Version != siteStateVersion {
		t.Fatalf("projection was not regenerated: changed=%v state=%#v", changed, state)
	}
}
