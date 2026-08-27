package clusterlog

import "testing"

func TestSlugifySpanishText(t *testing.T) {
	got := slugify("  Actualización del Clúster: Niño / ÁRBOL  ")
	want := "actualizacion-del-cluster-nino-arbol"
	if got != want {
		t.Fatalf("slugify() = %q, want %q", got, want)
	}
}

func TestParseDurationMinutes(t *testing.T) {
	tests := map[string]int{
		"90":    90,
		"90m":   90,
		"2h":    120,
		"1h30m": 90,
	}
	for input, want := range tests {
		got, err := parseDurationMinutes(input)
		if err != nil {
			t.Fatalf("parseDurationMinutes(%q): %v", input, err)
		}
		if got != want {
			t.Errorf("parseDurationMinutes(%q) = %d, want %d", input, got, want)
		}
	}
	if _, err := parseDurationMinutes("-5m"); err == nil {
		t.Fatal("expected negative duration to fail")
	}
}

func TestScanSecrets(t *testing.T) {
	text := "password = supersecreto\n-----BEGIN OPENSSH PRIVATE KEY-----"
	findings := scanSecrets(text)
	if !contains(findings, "contraseña explícita") || !contains(findings, "llave privada OpenSSH") {
		t.Fatalf("unexpected findings: %#v", findings)
	}
	if findings := scanSecrets("kubectl get nodes\nshow version"); len(findings) != 0 {
		t.Fatalf("false positive findings: %#v", findings)
	}
}

func TestPublicKeyFingerprintIsStable(t *testing.T) {
	key := testOpenSSHPublicKey(7, "ana@test")
	canonical1, fingerprint1, err := parsePublicKeyLine(key)
	if err != nil {
		t.Fatal(err)
	}
	canonical2, fingerprint2, err := parsePublicKeyLine("  " + key + "  ")
	if err != nil {
		t.Fatal(err)
	}
	if canonical1 != canonical2 || fingerprint1 != fingerprint2 {
		t.Fatalf("key parsing is not stable: %q/%q vs %q/%q", canonical1, fingerprint1, canonical2, fingerprint2)
	}
}
