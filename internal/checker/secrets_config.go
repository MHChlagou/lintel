package checker

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MHChlagou/lintel/internal/finding"
)

// Maximum bytes we'll read from any single config file. Anything larger is
// almost certainly a generated artifact, lockfile, or data blob — not a place
// humans hand-write secrets.
const maxConfigFileBytes = 1 * 1024 * 1024

// scanConfigSecrets looks for hardcoded credentials in declarative / config
// files where entropy-based scanners (gitleaks) deliberately back off. It
// covers Dockerfiles, .env, Java properties, docker-compose, Kubernetes and
// Helm manifests, Ansible playbooks/vars, Terraform / HCL, Spring / generic
// YAML configs, CI configs, TOML / INI / CFG / CONF, and JSON configs.
//
// Gitleaks optimizes for precision on source code; this scanner optimizes for
// catching the "obvious forgot-to-remove" mistake in configuration, even when
// the value is short or low-entropy. Deciding what "looks like a secret" is
// delegated to shouldFlagAssignment below.
//
// On pre-commit it reads the staged blob (consistent with `gitleaks protect
// --staged`); on pre-push / full scans it reads the HEAD blob.
func scanConfigSecrets(ctx context.Context, in CheckInput) ([]finding.Finding, error) {
	var out []finding.Finding
	for _, rel := range in.StagedFiles {
		kind, ok := isConfigSecretTarget(rel)
		if !ok {
			continue
		}
		content, err := readStagedOrWorking(ctx, in.RepoRoot, rel, in.Hook)
		if err != nil || len(content) == 0 || len(content) > maxConfigFileBytes {
			continue
		}
		out = append(out, scanConfigLines(rel, kind, content)...)
	}
	return out, nil
}

// noisyJSONBasenames lists JSON filenames that are overwhelmingly not secret
// carriers but do contain fields named things like "integrity" or "_token"
// that would trip heuristics. Skipping them keeps false positives down.
var noisyJSONBasenames = map[string]bool{
	"package.json":      true,
	"package-lock.json": true,
	"yarn.lock":         true, // not JSON but named explicitly
	"composer.lock":     true,
	"composer.json":     true,
	"tsconfig.json":     true,
}

// isConfigSecretTarget returns the kind of config file this path represents,
// or ok=false to skip. Scope is broad by design: lintel's promise is to catch
// forgotten secrets regardless of the file format. shouldFlagAssignment is
// what actually filters noise — path filtering just decides what we read.
//
// Explicitly excludes *.example / *.sample / *.template variants — those are
// expected to contain placeholder values, so flagging them would train users
// to ignore the scanner.
func isConfigSecretTarget(path string) (kind string, ok bool) {
	base := filepath.Base(path)
	lower := strings.ToLower(base)

	if hasAnySuffix(lower, ".example", ".sample", ".template", ".tmpl", ".dist") {
		return "", false
	}

	// ---- by exact basename ----
	switch lower {
	case ".env":
		return "dotenv", true
	case "dockerfile":
		return "dockerfile", true
	case "jenkinsfile":
		return "jenkinsfile", true
	case "ansible.cfg":
		return "ansible", true
	}

	// ---- by prefix (compound filenames) ----
	if strings.HasPrefix(lower, ".env.") {
		return "dotenv", true
	}
	if strings.HasPrefix(lower, "dockerfile.") {
		return "dockerfile", true
	}
	if (strings.HasPrefix(lower, "docker-compose") || strings.HasPrefix(lower, "compose")) &&
		(strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml")) {
		return "compose", true
	}

	// ---- by extension ----
	ext := strings.ToLower(filepath.Ext(lower))
	switch ext {
	case ".dockerfile":
		return "dockerfile", true
	case ".properties":
		return "properties", true
	case ".toml":
		return "toml", true
	case ".ini", ".cfg", ".conf":
		return "ini", true
	case ".tf", ".tfvars":
		return "terraform", true
	case ".hcl":
		return "hcl", true
	case ".yaml", ".yml":
		// Covers Kubernetes/Helm, Ansible, Spring, CI configs, and any other
		// declarative YAML. The policy function filters per-line.
		return yamlKind(path), true
	case ".json":
		if noisyJSONBasenames[lower] {
			return "", false
		}
		return "json", true
	}
	return "", false
}

// yamlKind labels a YAML file with a best-guess category based on its path.
// Used only for the human-readable "kind" in findings — not for filtering.
func yamlKind(path string) string {
	p := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(p, "/.github/workflows/"),
		strings.HasSuffix(p, ".gitlab-ci.yml"),
		strings.Contains(p, "/.circleci/"),
		strings.HasSuffix(p, "bitbucket-pipelines.yml"),
		strings.HasSuffix(p, "azure-pipelines.yml"):
		return "ci-config"
	case strings.Contains(p, "/roles/"),
		strings.Contains(p, "/playbooks/"),
		strings.Contains(p, "/group_vars/"),
		strings.Contains(p, "/host_vars/"),
		strings.Contains(p, "/inventory/"):
		return "ansible"
	case strings.Contains(p, "/k8s/"),
		strings.Contains(p, "/kubernetes/"),
		strings.Contains(p, "/manifests/"),
		strings.Contains(p, "/charts/"),
		strings.Contains(p, "/helm/"):
		return "kubernetes"
	}
	return "yaml"
}

func hasAnySuffix(s string, suffixes ...string) bool {
	for _, suf := range suffixes {
		if strings.HasSuffix(s, suf) {
			return true
		}
	}
	return false
}

// readStagedOrWorking returns the staged blob of `rel` on pre-commit (to match
// gitleaks `--staged` behavior), or the HEAD blob otherwise.
func readStagedOrWorking(ctx context.Context, repoRoot, rel, hook string) ([]byte, error) {
	if hook == "pre-commit" {
		cmd := exec.CommandContext(ctx, "git", "show", ":"+rel)
		cmd.Dir = repoRoot
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			return nil, err
		}
		return stdout.Bytes(), nil
	}
	cmd := exec.CommandContext(ctx, "git", "show", "HEAD:"+rel)
	cmd.Dir = repoRoot
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		return stdout.Bytes(), nil
	}
	return nil, nil
}

// assignmentLine captures a generic `KEY = VALUE` or `KEY : VALUE` assignment
// across shapes we care about:
//
//   - bare keys with dots/dashes (Java properties, TOML, Spring YAML): db.password=x
//   - quoted keys (JSON, YAML with quoted keys): "db_password": "x"
//   - optional leading directive: ENV/ARG/LABEL (Dockerfile), export (shell),
//     or a YAML list dash
//   - trailing commas (JSON)
//
// Groups: 1=double-quoted key, 2=single-quoted key, 3=bare key, 4=value.
// Exactly one of groups 1..3 is non-empty on a match.
var assignmentLine = regexp.MustCompile(
	`^\s*(?:-\s+|ENV\s+|ARG\s+|LABEL\s+|export\s+)?(?:"([^"]+)"|'([^']+)'|([A-Za-z_][A-Za-z0-9_.\-]*))\s*[:=]\s*(\S.*?)\s*,?\s*$`,
)

// k8sEnvWindow is how many lines we scan forward after a `name: FOO` before
// we give up pairing it with a `value: ...`. Kubernetes pods place them on
// adjacent lines; 5 gives headroom for comments/blank lines.
const k8sEnvWindow = 5

func scanConfigLines(path, kind string, content []byte) []finding.Finding {
	var findings []finding.Finding
	sc := bufio.NewScanner(bytes.NewReader(content))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Tracks the most recent `name: FOO` to pair with a later `value: BAR`
	// in Kubernetes / Helm env-list shape:
	//
	//   env:
	//   - name: DB_PASSWORD
	//     value: "Admin124"
	var pendingName string
	var pendingNameLine int

	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		m := assignmentLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key := firstNonEmpty(m[1], m[2], m[3])
		value := stripInlineComment(stripQuotes(m[4]))

		// Direct assignment check.
		if ruleID, ok := shouldFlagAssignment(key, value); ok {
			findings = append(findings, finding.Finding{
				Check:    "secrets",
				RuleID:   ruleID,
				Severity: finding.SevHigh,
				File:     path,
				Line:     lineNum,
				Column:   strings.Index(line, key) + 1,
				Message:  "Hardcoded credential in " + kind + " config (key=" + key + ")",
				Snippet:  redact(value),
				Engine:   "lintel-config",
			})
		}

		// Kubernetes env-list pairing: remember `name: X`, flag a subsequent
		// `value: Y` against the recorded name.
		lowerKey := strings.ToLower(key)
		if lowerKey == "name" && isSimpleIdentifier(value) {
			pendingName = value
			pendingNameLine = lineNum
			continue
		}
		if lowerKey == "value" && pendingName != "" && lineNum-pendingNameLine <= k8sEnvWindow {
			if ruleID, ok := shouldFlagAssignment(pendingName, value); ok {
				findings = append(findings, finding.Finding{
					Check:    "secrets",
					RuleID:   ruleID,
					Severity: finding.SevHigh,
					File:     path,
					Line:     lineNum,
					Column:   strings.Index(line, "value") + 1,
					Message:  "Hardcoded credential in " + kind + " config (env var " + pendingName + ")",
					Snippet:  redact(value),
					Engine:   "lintel-config",
				})
			}
			pendingName = ""
		}
	}
	return findings
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}

func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// stripInlineComment drops a `# ...` or `// ...` trailer from an already-dequoted
// value. Only runs post-dequote so we don't eat a legitimate `#` inside quotes.
func stripInlineComment(v string) string {
	if i := strings.Index(v, " #"); i >= 0 {
		v = v[:i]
	}
	if i := strings.Index(v, " //"); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}

// isSimpleIdentifier checks whether a value looks like a plain env-var name
// (letters, digits, underscores). Used to avoid pairing `name: "Some friendly
// description"` with the next `value:` line.
func isSimpleIdentifier(v string) bool {
	if v == "" {
		return false
	}
	for _, r := range v {
		if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// secretKeywords is the set of variable-name endings that signal "this holds
// a credential." Matched case-insensitively, either as the whole key or as
// an underscore-bounded suffix, so DB_PASSWORD, awsAccessKey (via camel-split
// normalization), spring.datasource.password, and STRIPE_API_KEY all hit.
var secretKeywords = []string{
	"password", "passwd", "passphrase", "pwd", "pass",
	"secret", "token",
	"credential", "credentials",
	"api_key", "apikey",
	"access_key", "accesskey",
	"private_key", "privatekey", "priv_key",
	"auth", "authorization", "bearer",
}

// placeholderValues are literal values that look like a secret-holding key but
// are obviously not a real secret. Compared lowercased, post-dequote.
var placeholderValues = map[string]bool{
	"changeme": true, "change_me": true, "change-me": true,
	"xxx": true, "xxxx": true, "xxxxx": true,
	"your-password": true, "your_password": true, "yourpassword": true,
	"your-secret": true, "your_secret": true, "yoursecret": true,
	"null": true, "none": true, "nil": true,
	"todo": true, "tbd": true, "example": true,
	"placeholder": true, "sample": true,
	"redacted": true, "hidden": true, "masked": true,
	"...": true, "***": true, "****": true, "*****": true,
}

// camelCaseBoundary splits `awsAccessKey` into `aws_Access_Key` so the suffix
// rules below apply uniformly to snake_case, camelCase, and PascalCase keys.
var camelCaseBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// shouldFlagAssignment decides whether a parsed KEY=VALUE from a declarative
// config file (Dockerfile, .env, properties, YAML, TOML, JSON, Terraform, …)
// represents a hardcoded credential worth blocking.
//
// Entropy is deliberately NOT used: gitleaks already owns entropy-based
// detection for source code, and the whole reason this scanner exists is to
// catch short, dumb passwords like `Admin124` that gitleaks ignores.
func shouldFlagAssignment(key, value string) (ruleID string, ok bool) {
	// Normalize the key so one suffix rule covers every casing/separator style.
	nk := camelCaseBoundary.ReplaceAllString(key, "${1}_${2}")
	nk = strings.ToLower(nk)
	nk = strings.ReplaceAll(nk, ".", "_")
	nk = strings.ReplaceAll(nk, "-", "_")

	var matched string
	for _, kw := range secretKeywords {
		if nk == kw || strings.HasSuffix(nk, "_"+kw) {
			matched = kw
			break
		}
	}
	if matched == "" {
		return "", false
	}

	v := strings.TrimSpace(value)
	if len(v) < 3 {
		return "", false
	}

	// Env-var interpolation and vault refs — the real secret is elsewhere.
	if strings.HasPrefix(v, "${") ||
		strings.HasPrefix(v, "$(") ||
		strings.HasPrefix(v, "%(") ||
		strings.HasPrefix(v, "!vault") ||
		v == "<no value>" {
		return "", false
	}
	// Single-dollar shell form: $FOO, $_BAR. Guard against $1.50 and $ alone.
	if len(v) >= 2 && v[0] == '$' {
		c := v[1]
		if c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			return "", false
		}
	}
	// Bracketed template placeholders: <password>, [secret], {{.Password}}.
	if (strings.HasPrefix(v, "<") && strings.HasSuffix(v, ">")) ||
		(strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]")) ||
		(strings.HasPrefix(v, "{{") && strings.HasSuffix(v, "}}")) {
		return "", false
	}
	if placeholderValues[strings.ToLower(v)] {
		return "", false
	}
	return "lintel.config-secret." + matched, true
}
