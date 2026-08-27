package clusterlog

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func testProjectRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWriteTestFile(t, filepath.Join(root, "zola.toml"), "title = \"test\"\n")
	for _, section := range []string{"documentacion", "manuales", "memorias", "proyectos", "tareas", "revision"} {
		if err := os.MkdirAll(filepath.Join(root, "content", section), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, dir := range []string{
		filepath.Join(root, "data", reviewEventsDirName),
		filepath.Join(root, "data", taskEventsDirName),
		filepath.Join(root, "config"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	registry := AdminRegistry{
		Version:      1,
		ReviewPolicy: ReviewPolicy{Default: "all-active", IncludeAuthor: true},
		Admins:       []Admin{},
	}
	if err := saveAdmins(root, registry); err != nil {
		t.Fatal(err)
	}
	return root
}

func mustWriteTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testOpenSSHPublicKey(seed byte, comment string) string {
	var blob bytes.Buffer
	writeSSHString := func(value []byte) {
		_ = binary.Write(&blob, binary.BigEndian, uint32(len(value)))
		_, _ = blob.Write(value)
	}
	writeSSHString([]byte("ssh-ed25519"))
	key := bytes.Repeat([]byte{seed}, 32)
	writeSSHString(key)
	encoded := base64.StdEncoding.EncodeToString(blob.Bytes())
	if comment == "" {
		return "ssh-ed25519 " + encoded
	}
	return "ssh-ed25519 " + encoded + " " + comment
}

func testAdmin(t *testing.T, id string, seed byte, roles ...string) Admin {
	t.Helper()
	publicKey := testOpenSSHPublicKey(seed, id+"@test")
	_, fingerprint, err := parsePublicKeyLine(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	return Admin{
		ID: id, Name: id, Email: id + "@example.test", Active: true, Roles: roles,
		SSHKeys: []SSHKey{{Fingerprint: fingerprint, PublicKey: publicKey, AddedAt: "2026-08-12T00:00:00Z"}},
	}
}
