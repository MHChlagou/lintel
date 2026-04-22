// Package config parses and validates .aegis/aegis.yaml.
package config

type Mode string

const (
	ModeBlock Mode = "block"
	ModeWarn  Mode = "warn"
	ModeOff   Mode = "off"
	ModeFix   Mode = "fix"
	ModeCheck Mode = "check"
)

// Spec mirrors aegis.yaml. Unknown fields are tolerated by yaml.v3 default settings.
type Spec struct {
	Version     int                 `yaml:"version"`
	Project     Project             `yaml:"project"`
	Binaries    map[string]Binary   `yaml:"binaries"`
	Checks      Checks              `yaml:"checks"`
	Scope       Scope               `yaml:"scope"`
	Hooks       map[string]HookSpec `yaml:"hooks"`
	Output      Output              `yaml:"output"`
	Override    Override            `yaml:"override"`
	Performance Performance         `yaml:"performance"`
	StrictVers  bool                `yaml:"strict_versions"`
}

type Project struct {
	Name      string   `yaml:"name"`
	Type      any      `yaml:"type"` // "auto" | []string
	Manifests []string `yaml:"manifests"`
}

type Binary struct {
	Command     string            `yaml:"command"`
	Path        string            `yaml:"path"`
	Version     string            `yaml:"version"`
	SHA256      map[string]string `yaml:"sha256"`
	InstallHint string            `yaml:"install_hint"`
}

type Checks struct {
	Secrets       SecretsCheck       `yaml:"secrets"`
	MaliciousCode MaliciousCodeCheck `yaml:"malicious_code"`
	Dependencies  DependenciesCheck  `yaml:"dependencies"`
	Lint          LintCheck          `yaml:"lint"`
	Format        FormatCheck        `yaml:"format"`
}

type ScanOpts struct {
	StagedOnly bool `yaml:"staged_only"`
	FullOnPush bool `yaml:"full_on_push"`
}

type SecretsCheck struct {
	Enabled          bool     `yaml:"enabled"`
	Engine           string   `yaml:"engine"`
	Mode             Mode     `yaml:"mode"`
	Scan             ScanOpts `yaml:"scan"`
	Rules            string   `yaml:"rules"`
	WarnPaths        []string `yaml:"warn_paths"`
	InlineIgnore     string   `yaml:"inline_ignore"`
	EntropyThreshold float64  `yaml:"entropy_threshold"`
}

type MaliciousCodeCheck struct {
	Enabled           bool     `yaml:"enabled"`
	Engine            string   `yaml:"engine"`
	Mode              Mode     `yaml:"mode"`
	Rulesets          []string `yaml:"rulesets"`
	SeverityThreshold string   `yaml:"severity_threshold"`
	TimeoutSeconds    int      `yaml:"timeout_seconds"`
	ExcludePaths      []string `yaml:"exclude_paths"`
}

type OfflineOpts struct {
	Enabled      bool   `yaml:"enabled"`
	RefreshHours int    `yaml:"refresh_hours"`
	DBPath       string `yaml:"db_path"`
}

type IgnoreCVE struct {
	ID      string `yaml:"id"`
	Reason  string `yaml:"reason"`
	Expires string `yaml:"expires"`
}

type DependenciesCheck struct {
	Enabled       bool        `yaml:"enabled"`
	Engine        string      `yaml:"engine"`
	Mode          Mode        `yaml:"mode"`
	BlockSeverity []string    `yaml:"block_severity"`
	SuggestFix    bool        `yaml:"suggest_fix"`
	Offline       OfflineOpts `yaml:"offline"`
	IgnoreCVEs    []IgnoreCVE `yaml:"ignore_cves"`
	ManifestGlobs []string    `yaml:"manifest_globs"`
}

type LintCheck struct {
	Enabled        bool                `yaml:"enabled"`
	Mode           Mode                `yaml:"mode"`
	AutoFix        bool                `yaml:"auto_fix"`
	FailOnSeverity string              `yaml:"fail_on_severity"`
	Tools          map[string]string   `yaml:"tools"`
	Args           map[string][]string `yaml:"args"`
}

type FormatCheck struct {
	Enabled bool              `yaml:"enabled"`
	Mode    Mode              `yaml:"mode"`
	Tools   map[string]string `yaml:"tools"`
}

type Scope struct {
	StagedOnly   bool     `yaml:"staged_only"`
	FullScanFor  []string `yaml:"full_scan_for"`
	ExcludePaths []string `yaml:"exclude_paths"`
}

type HookSpec struct {
	Enabled  *bool    `yaml:"enabled"`
	Checks   []string `yaml:"checks"`
	FailFast bool     `yaml:"fail_fast"`
}

func (h HookSpec) IsEnabled() bool {
	if h.Enabled != nil {
		return *h.Enabled
	}
	return len(h.Checks) > 0
}

type Output struct {
	Format             string `yaml:"format"`
	GroupBy            string `yaml:"group_by"`
	ShowFixSuggestions bool   `yaml:"show_fix_suggestions"`
	Color              string `yaml:"color"`
	Verbosity          string `yaml:"verbosity"`
	ReportFile         string `yaml:"report_file"`
}

type Override struct {
	EnvVar         string `yaml:"env_var"`
	AllowNoVerify  bool   `yaml:"allow_no_verify"`
	RequireReason  bool   `yaml:"require_reason"`
	LogFile        string `yaml:"log_file"`
	ProtectSecrets bool   `yaml:"protect_secrets"`
}

type CacheOpts struct {
	Enabled  bool   `yaml:"enabled"`
	Path     string `yaml:"path"`
	TTLHours int    `yaml:"ttl_hours"`
}

type Performance struct {
	Parallel            any       `yaml:"parallel"` // "auto" | int
	CheckTimeoutSeconds int       `yaml:"check_timeout_seconds"`
	TotalTimeoutSeconds int       `yaml:"total_timeout_seconds"`
	Cache               CacheOpts `yaml:"cache"`
}
