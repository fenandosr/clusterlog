package clusterlog

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func loadJSON(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("JSON inválido en %s: %w", path, err)
	}
	return nil
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data, 0o644)
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".clusterlog-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func readAll(path string, stdin io.Reader) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	if path == "-" {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(path)
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func bytesSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func generateID(prefix string, now time.Time) (string, error) {
	random := make([]byte, 3)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%s", strings.ToUpper(prefix), now.UTC().Format("20060102"), strings.ToUpper(hex.EncodeToString(random))), nil
}

func normalizeID(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func slugify(value string) string {
	replacements := map[rune]rune{
		'á': 'a', 'à': 'a', 'ä': 'a', 'â': 'a', 'Á': 'a', 'À': 'a', 'Ä': 'a', 'Â': 'a',
		'é': 'e', 'è': 'e', 'ë': 'e', 'ê': 'e', 'É': 'e', 'È': 'e', 'Ë': 'e', 'Ê': 'e',
		'í': 'i', 'ì': 'i', 'ï': 'i', 'î': 'i', 'Í': 'i', 'Ì': 'i', 'Ï': 'i', 'Î': 'i',
		'ó': 'o', 'ò': 'o', 'ö': 'o', 'ô': 'o', 'Ó': 'o', 'Ò': 'o', 'Ö': 'o', 'Ô': 'o',
		'ú': 'u', 'ù': 'u', 'ü': 'u', 'û': 'u', 'Ú': 'u', 'Ù': 'u', 'Ü': 'u', 'Û': 'u',
		'ñ': 'n', 'Ñ': 'n', 'ç': 'c', 'Ç': 'c',
	}
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if replacement, ok := replacements[r]; ok {
			r = replacement
		}
		r = unicode.ToLower(r)
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	seen := map[string]bool{}
	out := []string{}
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func parseDurationMinutes(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	if n, err := strconv.Atoi(value); err == nil {
		if n < 0 {
			return 0, fmt.Errorf("la duración no puede ser negativa")
		}
		return n, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("duración inválida %q; use minutos, 90m, 2h o 1h30m", value)
	}
	if duration < 0 {
		return 0, fmt.Errorf("la duración no puede ser negativa")
	}
	return int(duration.Round(time.Minute) / time.Minute), nil
}

func formatMinutes(minutes int) string {
	if minutes <= 0 {
		return "0m"
	}
	hours := minutes / 60
	rest := minutes % 60
	if hours == 0 {
		return fmt.Sprintf("%dm", rest)
	}
	if rest == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, rest)
}

func listMarkdownFiles(root, section string) ([]string, error) {
	base := filepath.Join(root, "content", section)
	entries := []string{}
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" || filepath.Base(path) == "_index.md" {
			return nil
		}
		entries = append(entries, path)
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	sort.Strings(entries)
	return entries, err
}

func sourceDigest(paths []string) (string, error) {
	sort.Strings(paths)
	h := sha256.New()
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		_, _ = h.Write([]byte(filepath.ToSlash(path)))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(data)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = true
		}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func hasRole(admin Admin, role string) bool {
	return contains(admin.Roles, role)
}

func relativePaths(root string, paths []string) ([]string, error) {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil, err
		}
		out = append(out, filepath.ToSlash(rel))
	}
	return out, nil
}

func validateDate(value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("fecha inválida %q; use YYYY-MM-DD", value)
	}
	return nil
}

var secretPatterns = []struct {
	Name    string
	Pattern *regexp.Regexp
}{
	{"llave privada OpenSSH", regexp.MustCompile(`(?i)BEGIN[ ]+OPENSSH[ ]+PRIVATE[ ]+KEY`)},
	{"llave privada PEM", regexp.MustCompile(`(?i)BEGIN[ ]+(RSA|EC|DSA)?[ ]*PRIVATE[ ]+KEY`)},
	{"token GitHub", regexp.MustCompile(`\b(ghp|github_pat)_[A-Za-z0-9_]{20,}\b`)},
	{"access key AWS", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{"token Slack", regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{20,}\b`)},
	{"contraseña explícita", regexp.MustCompile(`(?im)\b(password|passwd|contrase(?:ñ|n)a)\s*[:=]\s*[^<\s][^\n]{3,}`)},
}

func scanSecrets(text string) []string {
	findings := []string{}
	for _, item := range secretPatterns {
		if item.Pattern.MatchString(text) {
			findings = append(findings, item.Name)
		}
	}
	return uniqueSorted(findings)
}

func tomlString(value string) string { return strconv.Quote(value) }

func tomlStringArray(values []string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, tomlString(value))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func bullets(values []string) string {
	if len(values) == 0 {
		return "_No especificado._\n"
	}
	var b strings.Builder
	for _, value := range values {
		fmt.Fprintf(&b, "- %s\n", value)
	}
	return b.String()
}

func codeBlock(values []string) string {
	if len(values) == 0 {
		return "_No se registraron comandos._\n"
	}
	return "```console\n" + strings.Join(values, "\n") + "\n```\n"
}

func normalizeMarkdownBody(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	return strings.TrimSpace(body) + "\n"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func readLines(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func canonicalJSON(value any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
