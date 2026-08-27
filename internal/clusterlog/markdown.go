package clusterlog

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func parseMarkdown(path string) (PageMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PageMeta{}, err
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, "+++\n") {
		return PageMeta{}, fmt.Errorf("%s no tiene front matter TOML delimitado por +++", path)
	}
	rest := strings.TrimPrefix(text, "+++\n")
	idx := strings.Index(rest, "\n+++\n")
	if idx < 0 {
		if strings.HasSuffix(rest, "\n+++") {
			idx = len(rest) - len("\n+++")
		} else {
			return PageMeta{}, fmt.Errorf("%s tiene front matter sin cierre +++", path)
		}
	}
	rawFront := rest[:idx]
	bodyStart := idx + len("\n+++\n")
	body := ""
	if bodyStart <= len(rest) {
		body = rest[bodyStart:]
	}

	meta := PageMeta{
		Path:        path,
		Body:        body,
		Root:        map[string]any{},
		Taxonomies:  map[string][]string{},
		Extra:       map[string]any{},
		RawFront:    rawFront,
		ContentHash: bytesSHA256(data),
	}

	section := "root"
	scanner := bufio.NewScanner(strings.NewReader(rawFront))
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(stripTOMLComment(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		raw = strings.TrimSpace(raw)
		value, err := parseTOMLValue(raw)
		if err != nil {
			return PageMeta{}, fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
		switch section {
		case "root":
			meta.Root[key] = value
		case "taxonomies":
			values, ok := value.([]string)
			if ok {
				meta.Taxonomies[key] = values
			}
		case "extra":
			meta.Extra[key] = value
		default:
			// El parser intencionalmente ignora secciones no usadas por clusterlog.
		}
	}
	if err := scanner.Err(); err != nil {
		return PageMeta{}, err
	}
	return meta, nil
}

func stripTOMLComment(line string) string {
	inString := false
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inString {
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if r == '#' && !inString {
			return line[:i]
		}
	}
	return line
}

func parseTOMLValue(raw string) (any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if strings.HasPrefix(raw, "\"") {
		value, err := strconv.Unquote(raw)
		if err != nil {
			return nil, fmt.Errorf("cadena TOML inválida: %w", err)
		}
		return value, nil
	}
	if raw == "true" {
		return true, nil
	}
	if raw == "false" {
		return false, nil
	}
	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		return parseTOMLStringArray(raw)
	}
	if n, err := strconv.Atoi(raw); err == nil {
		return n, nil
	}
	// Fechas TOML y otros escalares simples se conservan como texto.
	return raw, nil
}

func parseTOMLStringArray(raw string) ([]string, error) {
	content := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(raw, "["), "]"))
	if content == "" {
		return []string{}, nil
	}
	var values []string
	var current strings.Builder
	inString := false
	escaped := false
	flush := func() error {
		item := strings.TrimSpace(current.String())
		current.Reset()
		if item == "" {
			return nil
		}
		value, err := strconv.Unquote(item)
		if err != nil {
			return fmt.Errorf("arreglo TOML inválido: %w", err)
		}
		values = append(values, value)
		return nil
	}
	for _, r := range content {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' && inString {
			current.WriteRune(r)
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			current.WriteRune(r)
			continue
		}
		if r == ',' && !inString {
			if err := flush(); err != nil {
				return nil, err
			}
			continue
		}
		current.WriteRune(r)
	}
	if inString {
		return nil, fmt.Errorf("arreglo TOML con cadena sin cerrar")
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return values, nil
}

func pageSlug(meta PageMeta) string {
	if value := meta.String("slug"); value != "" {
		return value
	}
	return slugify(strings.TrimSuffix(filepathBase(meta.Path), ".md"))
}

func filepathBase(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}
