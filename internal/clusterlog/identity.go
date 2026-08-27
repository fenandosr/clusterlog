package clusterlog

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func loadAdmins(root string) (AdminRegistry, error) {
	var registry AdminRegistry
	if err := loadJSON(filepath.Join(root, "data", "admins.json"), &registry); err != nil {
		return registry, err
	}
	if registry.Version == 0 {
		registry.Version = 1
	}
	if registry.ReviewPolicy.Default == "" {
		registry.ReviewPolicy.Default = "all-active"
		registry.ReviewPolicy.IncludeAuthor = true
	}
	return registry, nil
}

func saveAdmins(root string, registry AdminRegistry) error {
	if err := writeJSONAtomic(filepath.Join(root, "data", "admins.json"), registry); err != nil {
		return err
	}
	return writeAllowedSigners(root, registry)
}

func writeAllowedSigners(root string, registry AdminRegistry) error {
	lines := []string{
		"# Archivo generado por clusterlog. Contiene únicamente llaves públicas.",
		"# principal namespaces=\"git\" tipo base64",
	}
	seen := map[string]bool{}
	for _, admin := range registry.Admins {
		if !admin.Active {
			continue
		}
		for _, key := range admin.SSHKeys {
			fields := strings.Fields(key.PublicKey)
			if len(fields) < 2 {
				continue
			}
			line := fmt.Sprintf("%s namespaces=\"git\" %s %s", admin.Email, fields[0], fields[1])
			if !seen[line] {
				lines = append(lines, line)
				seen[line] = true
			}
		}
	}
	sort.Strings(lines[2:])
	return writeFileAtomic(filepath.Join(root, "config", "allowed_signers"), []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

func parsePublicKeyLine(line string) (canonical, fingerprint string, err error) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 2 {
		return "", "", fmt.Errorf("la llave pública debe tener formato OpenSSH: tipo base64 [comentario]")
	}
	decoded, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil {
		return "", "", fmt.Errorf("base64 inválido en llave pública: %w", err)
	}
	sum := sha256.Sum256(decoded)
	fingerprint = "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:])
	canonical = fields[0] + " " + fields[1]
	if len(fields) > 2 {
		canonical += " " + strings.Join(fields[2:], " ")
	}
	return canonical, fingerprint, nil
}

func readPublicKey(path string) (canonical, fingerprint, publicPath, signingPath string, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", "", "", "", fmt.Errorf("ruta de llave vacía")
	}
	expanded, err := expandPath(path)
	if err != nil {
		return "", "", "", "", err
	}
	original := expanded
	// user.signingkey suele apuntar a la llave privada; para identificar al
	// administrador siempre leemos la contraparte pública cuando existe.
	if !strings.HasSuffix(expanded, ".pub") {
		if _, pubErr := os.Stat(expanded + ".pub"); pubErr == nil {
			expanded += ".pub"
		}
	}
	info, statErr := os.Stat(expanded)
	if statErr != nil {
		return "", "", "", "", statErr
	}
	if info.IsDir() {
		return "", "", "", "", fmt.Errorf("%s es un directorio", expanded)
	}
	data, err := os.ReadFile(expanded)
	if err != nil {
		return "", "", "", "", err
	}
	canonical, fingerprint, err = parsePublicKeyLine(string(data))
	if err != nil {
		return "", "", "", "", err
	}
	publicPath = expanded
	signingPath = expanded
	if original != expanded {
		signingPath = original
	} else if strings.HasSuffix(expanded, ".pub") {
		privateCandidate := strings.TrimSuffix(expanded, ".pub")
		if _, err := os.Stat(privateCandidate); err == nil {
			signingPath = privateCandidate
		}
	}
	return canonical, fingerprint, publicPath, signingPath, nil
}

func expandPath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return filepath.Abs(path)
}

func candidateKeyPaths(explicit string) []string {
	candidates := []string{}
	if explicit != "" {
		candidates = append(candidates, explicit)
	}
	if env := os.Getenv("CLUSTERLOG_SSH_KEY"); env != "" {
		candidates = append(candidates, env)
	}
	if output, err := exec.Command("git", "config", "--get", "user.signingkey").Output(); err == nil {
		value := strings.TrimSpace(string(output))
		if strings.HasPrefix(value, "key::") {
			// Una llave inline se identifica, pero no sirve como ruta para firmar.
		} else if value != "" {
			candidates = append(candidates, value)
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		for _, name := range []string{"id_ed25519.pub", "id_ecdsa.pub", "id_rsa.pub"} {
			candidates = append(candidates, filepath.Join(home, ".ssh", name))
		}
	}
	seen := map[string]bool{}
	out := []string{}
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		out = append(out, candidate)
	}
	return out
}

func (a *App) ResolveIdentity(root string) (Identity, error) {
	registry, err := loadAdmins(root)
	if err != nil {
		return Identity{}, err
	}
	attempted := []string{}
	for _, path := range candidateKeyPaths(a.Options.SSHKey) {
		canonical, fingerprint, publicPath, signingPath, err := readPublicKey(path)
		if err != nil {
			attempted = append(attempted, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		for _, admin := range registry.Admins {
			if !admin.Active {
				continue
			}
			for _, key := range admin.SSHKeys {
				if key.Fingerprint == fingerprint {
					return Identity{
						Admin: admin, Fingerprint: fingerprint, PublicKey: canonical,
						PublicPath: publicPath, SigningKey: signingPath,
					}, nil
				}
			}
		}
		attempted = append(attempted, fmt.Sprintf("%s: %s no está registrada", path, fingerprint))
	}
	return Identity{}, NewError(ExitIdentity, "identity_not_found",
		"ninguna llave SSH local coincide con un administrador activo", map[string]any{
			"attempted": attempted,
			"hint":      "use --ssh-key ~/.ssh/id_ed25519.pub o CLUSTERLOG_SSH_KEY",
		})
}

func adminByID(registry AdminRegistry, id string) (Admin, bool) {
	id = normalizeID(id)
	for _, admin := range registry.Admins {
		if normalizeID(admin.ID) == id {
			return admin, true
		}
	}
	return Admin{}, false
}

func activeAdminIDs(registry AdminRegistry) []string {
	ids := []string{}
	for _, admin := range registry.Admins {
		if admin.Active {
			ids = append(ids, admin.ID)
		}
	}
	sort.Strings(ids)
	return ids
}
