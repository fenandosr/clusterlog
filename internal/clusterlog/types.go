package clusterlog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Version se puede sobreescribir en tiempo de build con:
//
//	-ldflags "-X example.org/bitacora-cluster/internal/clusterlog.Version=v0.2.0"
//
// Ver la variable CLUSTERLOG_VERSION en el Makefile.
var Version = "0.1.0"

const (
	ExitOK         = 0
	ExitUsage      = 2
	ExitIdentity   = 3
	ExitValidation = 4
	ExitConflict   = 5
	ExitGit        = 6
	ExitInternal   = 10
)

type CLIError struct {
	Code    int
	Kind    string
	Message string
	Details any
}

func (e *CLIError) Error() string { return e.Message }

func NewError(code int, kind, message string, details any) *CLIError {
	return &CLIError{Code: code, Kind: kind, Message: message, Details: details}
}

type Options struct {
	Root       string
	JSON       bool
	SSHKey     string
	Commit     bool
	DryRun     bool
	Stdout     io.Writer
	Stderr     io.Writer
	Stdin      io.Reader
	Now        func() time.Time
	Executable string
}

type App struct {
	Options Options
}

func New(options Options) *App {
	if options.Stdout == nil {
		options.Stdout = os.Stdout
	}
	if options.Stderr == nil {
		options.Stderr = os.Stderr
	}
	if options.Stdin == nil {
		options.Stdin = os.Stdin
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Executable == "" {
		options.Executable = "clusterlog"
	}
	return &App{Options: options}
}

type ReviewPolicy struct {
	Default       string `json:"default"`
	IncludeAuthor bool   `json:"include_author"`
}

type SSHKey struct {
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"public_key"`
	AddedAt     string `json:"added_at,omitempty"`
}

type Admin struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Active  bool     `json:"active"`
	Roles   []string `json:"roles,omitempty"`
	SSHKeys []SSHKey `json:"ssh_keys"`
}

type AdminRegistry struct {
	Version      int          `json:"version"`
	ReviewPolicy ReviewPolicy `json:"review_policy"`
	Admins       []Admin      `json:"admins"`
}

type Identity struct {
	Admin       Admin  `json:"admin"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"public_key"`
	PublicPath  string `json:"public_key_path"`
	SigningKey  string `json:"signing_key_path"`
}

type ReviewEvent struct {
	ID            string `json:"id"`
	EntryID       string `json:"entry_id"`
	AdminID       string `json:"admin_id"`
	Fingerprint   string `json:"fingerprint"`
	ReviewedAt    string `json:"reviewed_at"`
	ContentSHA256 string `json:"content_sha256"`
	Note          string `json:"note,omitempty"`
}

type ReviewLog struct {
	Version int           `json:"version"`
	Events  []ReviewEvent `json:"events"`
}

type TaskEvent struct {
	ID          string `json:"id"`
	TaskID      string `json:"task_id"`
	Type        string `json:"type"`
	AdminID     string `json:"admin_id"`
	Fingerprint string `json:"fingerprint"`
	At          string `json:"at"`
	Status      string `json:"status,omitempty"`
	Minutes     int    `json:"minutes,omitempty"`
	Note        string `json:"note,omitempty"`
}

type TaskLog struct {
	Version int         `json:"version"`
	Events  []TaskEvent `json:"events"`
}

type PageMeta struct {
	Path        string
	Body        string
	Root        map[string]any
	Taxonomies  map[string][]string
	Extra       map[string]any
	RawFront    string
	ContentHash string
}

func (p PageMeta) String(key string) string {
	if v, ok := p.Root[key]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

func (p PageMeta) ExtraString(key string) string {
	if v, ok := p.Extra[key]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

func (p PageMeta) ExtraBool(key string) bool {
	v, ok := p.Extra[key]
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

func (p PageMeta) ExtraInt(key string) int {
	v, ok := p.Extra[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func (p PageMeta) ExtraStrings(key string) []string {
	v, ok := p.Extra[key]
	if !ok {
		return nil
	}
	switch values := v.(type) {
	case []string:
		return values
	case []any:
		out := make([]string, 0, len(values))
		for _, item := range values {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		return nil
	}
}

type MemoryRecord struct {
	EntryID           string   `json:"entry_id"`
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	Date              string   `json:"date"`
	AuthorID          string   `json:"author_id"`
	AuthorFingerprint string   `json:"author_fingerprint"`
	Risk              string   `json:"risk"`
	Ticket            string   `json:"ticket,omitempty"`
	ReviewRequired    bool     `json:"review_required"`
	ReviewPolicy      string   `json:"review_policy"`
	ReviewerSnapshot  bool     `json:"reviewer_snapshot"`
	Reviewers         []string `json:"reviewers"`
	Tags              []string `json:"tags"`
	Systems           []string `json:"systems"`
	Slug              string   `json:"slug"`
	Permalink         string   `json:"permalink"`
	SourcePath        string   `json:"source_path"`
	ContentSHA256     string   `json:"content_sha256"`
}

type ReviewQueueItem struct {
	EntryID            string   `json:"entry_id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Date               string   `json:"date"`
	AuthorID           string   `json:"author_id"`
	Risk               string   `json:"risk"`
	Tags               []string `json:"tags"`
	Systems            []string `json:"systems"`
	Permalink          string   `json:"permalink"`
	ContentSHA256      string   `json:"content_sha256"`
	ReviewedBy         []string `json:"reviewed_by"`
	MissingReviewers   []string `json:"missing_reviewers"`
	RequiredCount      int      `json:"required_count"`
	ReviewedCount      int      `json:"reviewed_count"`
	ConfigurationError string   `json:"configuration_error,omitempty"`
}

type TaskState struct {
	TaskID          string   `json:"task_id"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Status          string   `json:"status"`
	Priority        string   `json:"priority"`
	ProjectID       string   `json:"project_id,omitempty"`
	Assignees       []string `json:"assignees"`
	EstimateMinutes int      `json:"estimate_minutes"`
	ActualMinutes   int      `json:"actual_minutes"`
	Due             string   `json:"due,omitempty"`
	RunningBy       []string `json:"running_by"`
	Slug            string   `json:"slug"`
	Permalink       string   `json:"permalink"`
	SourcePath      string   `json:"source_path"`
}

type ProjectState struct {
	ProjectID   string   `json:"project_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Owners      []string `json:"owners"`
	TargetDate  string   `json:"target_date,omitempty"`
	Slug        string   `json:"slug"`
	Permalink   string   `json:"permalink"`
	SourcePath  string   `json:"source_path"`
}

type SiteStats struct {
	ActiveAdmins   int `json:"active_admins"`
	PendingReviews int `json:"pending_reviews"`
	OpenTasks      int `json:"open_tasks"`
	ActiveProjects int `json:"active_projects"`
}

type SiteState struct {
	Version        int               `json:"version"`
	GeneratedAt    string            `json:"generated_at"`
	SourceSHA256   string            `json:"source_sha256"`
	Stats          SiteStats         `json:"stats"`
	ReviewQueue    []ReviewQueueItem `json:"review_queue"`
	RecentMemories []MemoryRecord    `json:"recent_memories"`
	Tasks          []TaskState       `json:"tasks"`
	Projects       []ProjectState    `json:"projects"`
}

type ResultEnvelope struct {
	OK       bool     `json:"ok"`
	Command  string   `json:"command,omitempty"`
	Data     any      `json:"data,omitempty"`
	Error    any      `json:"error,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// marshalIndentNoEscape serializa igual que json.MarshalIndent pero sin el
// escape HTML por defecto de encoding/json (que convierte <, > y & en
// <, > y &). La salida de la CLI es para terminal/agentes,
// no para incrustar en HTML, así que ese escape sólo confunde cualquier
// texto de ayuda o dato que contenga esos caracteres (por ejemplo, los
// placeholders <id>/<correo> de bootstrap-instance).
func marshalIndentNoEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func (a *App) printResult(command string, data any, warnings ...string) error {
	if a.Options.JSON {
		bytes, err := marshalIndentNoEscape(ResultEnvelope{OK: true, Command: command, Data: data, Warnings: warnings})
		if err != nil {
			return err
		}
		fmt.Fprintln(a.Options.Stdout, string(bytes))
		return nil
	}
	switch v := data.(type) {
	case string:
		fmt.Fprintln(a.Options.Stdout, v)
	default:
		bytes, err := marshalIndentNoEscape(v)
		if err != nil {
			return err
		}
		fmt.Fprintln(a.Options.Stdout, string(bytes))
	}
	for _, warning := range warnings {
		fmt.Fprintf(a.Options.Stderr, "ADVERTENCIA: %s\n", warning)
	}
	return nil
}

func PrintError(w io.Writer, jsonMode bool, err error) int {
	var cliErr *CLIError
	if !errors.As(err, &cliErr) {
		cliErr = NewError(ExitInternal, "internal", err.Error(), nil)
	}
	if jsonMode {
		payload := ResultEnvelope{OK: false, Error: map[string]any{
			"kind":      cliErr.Kind,
			"message":   cliErr.Message,
			"details":   cliErr.Details,
			"exit_code": cliErr.Code,
		}}
		bytes, err := marshalIndentNoEscape(payload)
		if err == nil {
			fmt.Fprintln(w, string(bytes))
		}
	} else {
		fmt.Fprintf(w, "ERROR [%s]: %s\n", cliErr.Kind, cliErr.Message)
		if cliErr.Details != nil {
			bytes, _ := marshalIndentNoEscape(cliErr.Details)
			fmt.Fprintln(w, string(bytes))
		}
	}
	return cliErr.Code
}

func (a *App) ResolveRoot() (string, error) {
	if a.Options.Root != "" {
		root, err := filepath.Abs(a.Options.Root)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(filepath.Join(root, "zola.toml")); err != nil {
			return "", NewError(ExitUsage, "root_not_found", "el directorio raíz no contiene zola.toml", root)
		}
		return root, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	current := cwd
	for {
		if _, err := os.Stat(filepath.Join(current, "zola.toml")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", NewError(ExitUsage, "root_not_found", "no se encontró zola.toml en el directorio actual ni en sus padres", cwd)
}
