package clusterlog

import (
	"embed"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed all:scaffold
var scaffoldFS embed.FS

const scaffoldRoot = "scaffold"

var scaffoldSections = []string{"documentacion", "manuales", "memorias", "proyectos", "tareas", "revision", "topologia"}

// runBootstrapInstance genera el esqueleto de una instancia nueva de
// clusterlog (content/, data/, config/, zola.toml) en --output. No toca el
// proyecto actual: --output es un directorio nuevo, casi siempre fuera de
// este repositorio, así que este comando no pasa por ResolveRoot ni por el
// mecanismo de --commit.
func (a *App) runBootstrapInstance(args []string) error {
	fs := newFlagSet("bootstrap-instance")
	output := fs.String("output", "", "directorio donde crear la instancia nueva")
	productName := fs.String("product-name", "Bitácora Operativa", "nombre del producto para zola.toml")
	baseURL := fs.String("base-url", "https://clusterlog.example.org", "base_url de zola.toml; cámbielo antes de publicar")
	if err := fs.Parse(args); err != nil {
		return NewError(ExitUsage, "invalid_flags", err.Error(), nil)
	}
	if strings.TrimSpace(*output) == "" {
		return NewError(ExitUsage, "missing_output", "indique --output con el directorio de la instancia nueva", nil)
	}
	root, err := filepath.Abs(*output)
	if err != nil {
		return NewError(ExitInternal, "internal", err.Error(), nil)
	}

	if entries, statErr := os.ReadDir(root); statErr == nil {
		if len(entries) > 0 {
			return NewError(ExitConflict, "output_not_empty", "el directorio de salida ya existe y no está vacío", root)
		}
	} else if !os.IsNotExist(statErr) {
		return NewError(ExitInternal, "internal", statErr.Error(), root)
	}

	if a.Options.DryRun {
		return a.printResult("bootstrap-instance", map[string]any{
			"output":       root,
			"product_name": *productName,
			"base_url":     *baseURL,
			"dry_run":      true,
		})
	}

	created, err := writeScaffold(root, *productName, *baseURL)
	if err != nil {
		return err
	}

	registry := AdminRegistry{Version: 1, ReviewPolicy: ReviewPolicy{Default: "all-active", IncludeAuthor: true}, Admins: []Admin{}}
	if err := saveAdmins(root, registry); err != nil {
		return NewError(ExitInternal, "internal", err.Error(), nil)
	}
	created = append(created,
		filepath.Join(root, "data", "admins.json"),
		filepath.Join(root, "config", "allowed_signers"),
	)
	sort.Strings(created)

	return a.printResult("bootstrap-instance", map[string]any{
		"output":         root,
		"engine_version": Version,
		"product_name":   *productName,
		"base_url":       *baseURL,
		"files_created":  created,
		"next_steps": []string{
			"clusterlog --root " + root + " --ssh-key <su-llave.pub> admin add --id <id> --name <nombre> --email <correo> --key <su-llave.pub> --roles owner,admin",
			"clusterlog --root " + root + " sync",
			"clusterlog --root " + root + " validate --skip-zola",
		},
	})
}

func writeScaffold(root, productName, baseURL string) ([]string, error) {
	created := []string{}

	title := "Bitácora Operativa"
	if strings.TrimSpace(productName) != "" {
		title = productName
	}

	homeTmpl, err := scaffoldFS.ReadFile(scaffoldRoot + "/content/_index.md.tmpl")
	if err != nil {
		return nil, NewError(ExitInternal, "internal", "plantilla de portada faltante", err.Error())
	}
	homeContent := renderScaffoldTemplate(string(homeTmpl), map[string]string{"TITLE": title})
	homePath := filepath.Join(root, "content", "_index.md")
	if err := writeFileAtomic(homePath, []byte(homeContent), 0o644); err != nil {
		return nil, NewError(ExitInternal, "internal", err.Error(), homePath)
	}
	created = append(created, homePath)

	for _, section := range scaffoldSections {
		src := scaffoldRoot + "/content/" + section + "/_index.md"
		data, err := scaffoldFS.ReadFile(src)
		if err != nil {
			return nil, NewError(ExitInternal, "internal", "plantilla de scaffold faltante: "+src, err.Error())
		}
		dst := filepath.Join(root, "content", section, "_index.md")
		if err := writeFileAtomic(dst, data, 0o644); err != nil {
			return nil, NewError(ExitInternal, "internal", err.Error(), dst)
		}
		created = append(created, dst)
	}

	zolaTmpl, err := scaffoldFS.ReadFile(scaffoldRoot + "/zola.toml.tmpl")
	if err != nil {
		return nil, NewError(ExitInternal, "internal", "plantilla zola.toml faltante", err.Error())
	}
	zolaContent := renderScaffoldTemplate(string(zolaTmpl), map[string]string{
		"BASE_URL":     baseURL,
		"TITLE":        title,
		"PRODUCT_NAME": productName,
	})
	zolaPath := filepath.Join(root, "zola.toml")
	if err := writeFileAtomic(zolaPath, []byte(zolaContent), 0o644); err != nil {
		return nil, NewError(ExitInternal, "internal", err.Error(), zolaPath)
	}
	created = append(created, zolaPath)

	gitignoreTmpl, err := scaffoldFS.ReadFile(scaffoldRoot + "/gitignore.tmpl")
	if err != nil {
		return nil, NewError(ExitInternal, "internal", "plantilla .gitignore faltante", err.Error())
	}
	gitignorePath := filepath.Join(root, ".gitignore")
	if err := writeFileAtomic(gitignorePath, gitignoreTmpl, 0o644); err != nil {
		return nil, NewError(ExitInternal, "internal", err.Error(), gitignorePath)
	}
	created = append(created, gitignorePath)

	readmeTmpl, err := scaffoldFS.ReadFile(scaffoldRoot + "/README.md.tmpl")
	if err != nil {
		return nil, NewError(ExitInternal, "internal", "plantilla README faltante", err.Error())
	}
	readmeContent := renderScaffoldTemplate(string(readmeTmpl), map[string]string{
		"TITLE":        title,
		"PRODUCT_NAME": productName,
	})
	readmePath := filepath.Join(root, "README.md")
	if err := writeFileAtomic(readmePath, []byte(readmeContent), 0o644); err != nil {
		return nil, NewError(ExitInternal, "internal", err.Error(), readmePath)
	}
	created = append(created, readmePath)

	versionPath := filepath.Join(root, ".clusterlog-version")
	if err := writeFileAtomic(versionPath, []byte(Version+"\n"), 0o644); err != nil {
		return nil, NewError(ExitInternal, "internal", err.Error(), versionPath)
	}
	created = append(created, versionPath)

	templatesWritten, err := writeEmbeddedTemplates(filepath.Join(root, "templates"))
	if err != nil {
		return nil, err
	}
	created = append(created, templatesWritten...)

	for _, dir := range []string{filepath.Join(root, "data", "review-events"), filepath.Join(root, "data", "task-events")} {
		keep := filepath.Join(dir, ".gitkeep")
		if err := writeFileAtomic(keep, []byte{}, 0o644); err != nil {
			return nil, NewError(ExitInternal, "internal", err.Error(), keep)
		}
		created = append(created, keep)
	}

	if err := os.MkdirAll(filepath.Join(root, "data", "generated"), 0o755); err != nil {
		return nil, NewError(ExitInternal, "internal", err.Error(), nil)
	}

	return created, nil
}

func renderScaffoldTemplate(text string, values map[string]string) string {
	for key, value := range values {
		text = strings.ReplaceAll(text, "{{"+key+"}}", value)
	}
	return text
}
