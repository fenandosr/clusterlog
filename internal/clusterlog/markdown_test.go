package clusterlog

import (
	"path/filepath"
	"testing"
)

func TestParseMarkdownFrontMatter(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "page.md")
	mustWriteTestFile(t, path, `+++
title = "Prueba # literal"
description = "Página de prueba"
date = 2026-08-12T08:30:00Z
authors = ["ana", "bob"]
slug = "pagina-prueba"

[taxonomies]
tags = ["red", "cambio"]
systems = ["spine-01"]

[extra]
id = "MEM-TEST-001"
review_required = true
duration_minutes = 45
reviewers = ["ana", "bob"] # comentario
+++

## Cuerpo

Contenido.
`)
	meta, err := parseMarkdown(path)
	if err != nil {
		t.Fatal(err)
	}
	if meta.String("title") != "Prueba # literal" {
		t.Fatalf("unexpected title: %q", meta.String("title"))
	}
	if meta.ExtraString("id") != "MEM-TEST-001" || !meta.ExtraBool("review_required") || meta.ExtraInt("duration_minutes") != 45 {
		t.Fatalf("unexpected extra metadata: %#v", meta.Extra)
	}
	if got := meta.Taxonomies["tags"]; len(got) != 2 || got[0] != "red" || got[1] != "cambio" {
		t.Fatalf("unexpected tags: %#v", got)
	}
	if meta.ContentHash == "" || meta.Body == "" {
		t.Fatal("expected content hash and body")
	}
}

func TestParseMarkdownRejectsUnclosedFrontMatter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.md")
	mustWriteTestFile(t, path, "+++\ntitle = \"sin cierre\"\n")
	if _, err := parseMarkdown(path); err == nil {
		t.Fatal("expected invalid front matter to fail")
	}
}
