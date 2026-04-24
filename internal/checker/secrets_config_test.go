package checker

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestShouldFlagAssignment(t *testing.T) {
	cases := []struct {
		name     string
		key, val string
		wantFlag bool
		wantRule string
	}{
		// --- hits ---
		{"dockerfile env pass", "pass", "Admin124", true, "lintel.config-secret.pass"},
		{"env DB_PASSWORD", "DB_PASSWORD", "secret123", true, "lintel.config-secret.password"},
		{"spring dotted password", "spring.datasource.password", "hunter2", true, "lintel.config-secret.password"},
		{"camelCase awsAccessKey", "awsAccessKey", "AKIAIOSFODNN7EXAMPLE", true, "lintel.config-secret.access_key"},
		{"dashed my-app-secret", "my-app-secret", "opaque", true, "lintel.config-secret.secret"},
		{"generic api_key", "API_KEY", "ghp_abc", true, "lintel.config-secret.api_key"},
		{"auth bare", "auth", "basicBase64Val", true, "lintel.config-secret.auth"},
		{"bearer token var", "MY_BEARER", "token-xyz", true, "lintel.config-secret.bearer"},

		// --- near-misses the boundary rule should reject ---
		{"passport not pass", "passport", "abc123", false, ""},
		{"bypass not pass", "bypass", "abc123", false, ""},
		{"compass not pass", "compass", "abc123", false, ""},
		{"username irrelevant", "username", "hedi", false, ""},

		// --- placeholder / interpolation filters ---
		{"interpolation ${}", "password", "${DB_PASSWORD}", false, ""},
		{"interpolation $VAR", "password", "$DB_PASSWORD", false, ""},
		{"interpolation $(cmd)", "password", "$(cat secret)", false, ""},
		{"ansible vault", "password", "!vault |\n  $ANSIBLE_VAULT;1.1", false, ""},
		{"helm no value", "password", "<no value>", false, ""},
		{"templated <password>", "password", "<password>", false, ""},
		{"templated {{.Pw}}", "password", "{{.Password}}", false, ""},
		{"placeholder changeme", "password", "changeme", false, ""},
		{"placeholder your-password", "password", "your-password", false, ""},
		{"placeholder null", "password", "null", false, ""},
		{"too short", "password", "ab", false, ""},
		{"empty", "password", "", false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rule, ok := shouldFlagAssignment(c.key, c.val)
			if ok != c.wantFlag {
				t.Fatalf("flag: got %v, want %v (rule=%q)", ok, c.wantFlag, rule)
			}
			if c.wantFlag && rule != c.wantRule {
				t.Fatalf("rule: got %q, want %q", rule, c.wantRule)
			}
		})
	}
}

func TestIsConfigSecretTarget(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"Dockerfile", true},
		{"Dockerfile.prod", true},
		{"app/Dockerfile", true},
		{"prod.Dockerfile", true},
		{".env", true},
		{".env.production", true},
		{".env.example", false},
		{".env.sample", false},
		{"docker-compose.yml", true},
		{"compose.yaml", true},
		{"application.properties", true},
		{"config/config.toml", true},
		{"setup.cfg", true},
		{"k8s/deployment.yaml", true},
		{".github/workflows/ci.yml", true},
		{"main.tf", true},
		{"terraform.tfvars", true},
		{"Jenkinsfile", true},
		{"ansible.cfg", true},
		{"config.json", true},
		{"package.json", false},
		{"package-lock.json", false},
		{"tsconfig.json", false},
		{"src/main.go", false},
		{"README.md", false},
		{"index.html", false},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			_, ok := isConfigSecretTarget(c.path)
			if ok != c.want {
				t.Fatalf("isConfigSecretTarget(%q) = %v, want %v", c.path, ok, c.want)
			}
		})
	}
}

func TestScanConfigLines_Dockerfile(t *testing.T) {
	content := []byte(`FROM node:20
WORKDIR /app
ENV pass=Admin124
ENV PASSWORD="changeme"
ENV DB_PASSWORD=${SECRET_FROM_VAULT}
ENV API_KEY="sk_live_abc123def456"
# ENV commented=Admin124
CMD ["node", "index.js"]
`)
	got := scanConfigLines("Dockerfile", "dockerfile", content)
	if len(got) != 2 {
		t.Fatalf("want 2 findings, got %d: %+v", len(got), got)
	}
	if got[0].Line != 3 || got[0].RuleID != "lintel.config-secret.pass" {
		t.Errorf("first finding wrong: %+v", got[0])
	}
	if got[1].Line != 6 || got[1].RuleID != "lintel.config-secret.api_key" {
		t.Errorf("second finding wrong: %+v", got[1])
	}
}

func TestScanConfigLines_DotEnv(t *testing.T) {
	content := []byte(`# generated
DB_HOST=localhost
DB_PASSWORD=hunter2
API_KEY=changeme
PASSWORD=${FROM_VAULT}
`)
	got := scanConfigLines(".env", "dotenv", content)
	if len(got) != 1 || got[0].RuleID != "lintel.config-secret.password" || got[0].Line != 3 {
		t.Fatalf("want one DB_PASSWORD hit on line 3, got %+v", got)
	}
}

func TestScanConfigLines_SpringProperties(t *testing.T) {
	content := []byte(`spring.application.name=demo
spring.datasource.url=jdbc:postgres://db:5432/x
spring.datasource.password=hunter2
spring.datasource.username=admin
`)
	got := scanConfigLines("application.properties", "properties", content)
	if len(got) != 1 || got[0].RuleID != "lintel.config-secret.password" || got[0].Line != 3 {
		t.Fatalf("want one spring.datasource.password hit on line 3, got %+v", got)
	}
}

func TestScanConfigLines_KubernetesEnvPair(t *testing.T) {
	content := []byte(`apiVersion: v1
kind: Pod
spec:
  containers:
  - name: app
    env:
    - name: DB_PASSWORD
      value: "Admin124"
    - name: LOG_LEVEL
      value: info
`)
	got := scanConfigLines("k8s/pod.yaml", "kubernetes", content)
	if len(got) != 1 {
		t.Fatalf("want 1 finding from name/value pair, got %d: %+v", len(got), got)
	}
	if got[0].RuleID != "lintel.config-secret.password" {
		t.Errorf("rule id: %+v", got[0])
	}
	if got[0].Line != 8 {
		t.Errorf("line: got %d, want 8", got[0].Line)
	}
}

func TestScanConfigLines_JSONConfig(t *testing.T) {
	content := []byte(`{
  "service": "app",
  "apiKey": "abc123def456",
  "db_password": "hunter2"
}
`)
	got := scanConfigLines("config.json", "json", content)
	if len(got) != 2 {
		t.Fatalf("want 2 findings, got %d: %+v", len(got), got)
	}
}

// initRepoWithDockerfile builds a throwaway git repo containing a Dockerfile
// with a known secret, committed to HEAD. Returns the repo root.
func initRepoWithDockerfile(t *testing.T, dockerfileBody string) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	run("git", "init", "-q")
	run("git", "config", "user.email", "test@example.com")
	run("git", "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(dockerfileBody), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "Dockerfile")
	run("git", "commit", "-q", "-m", "initial")
	return dir
}

// TestScanConfigSecrets_PrePushFallback locks in the regression we discovered
// in CI: pre-push runs have an empty StagedFiles slice (nothing is staged
// when you push), so the scanner must fall back to the full HEAD tree or it
// silently reports no findings — exactly the silent-fail mode this whole
// scanner exists to prevent.
func TestScanConfigSecrets_PrePushFallback(t *testing.T) {
	root := initRepoWithDockerfile(t, "FROM node:20\nENV pass=Admin124\nCMD [\"node\"]\n")
	got, err := scanConfigSecrets(context.Background(), CheckInput{
		RepoRoot:    root,
		StagedFiles: nil, // pre-push: nothing staged
		Hook:        "pre-push",
	})
	if err != nil {
		t.Fatalf("scanConfigSecrets: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 finding from full-tree fallback, got %d: %+v", len(got), got)
	}
	if got[0].RuleID != "lintel.config-secret.pass" {
		t.Errorf("rule id: %+v", got[0])
	}
	if got[0].File != "Dockerfile" {
		t.Errorf("file: got %q, want Dockerfile", got[0].File)
	}
}

// TestScanConfigSecrets_PreCommitUsesStaged confirms we did not break the
// pre-commit path. Files committed to HEAD but not present in StagedFiles
// must be skipped, since pre-commit's contract is "scan the staged diff."
func TestScanConfigSecrets_PreCommitUsesStaged(t *testing.T) {
	root := initRepoWithDockerfile(t, "FROM node:20\nENV pass=Admin124\nCMD [\"node\"]\n")
	got, err := scanConfigSecrets(context.Background(), CheckInput{
		RepoRoot:    root,
		StagedFiles: nil, // pre-commit with nothing staged
		Hook:        "pre-commit",
	})
	if err != nil {
		t.Fatalf("scanConfigSecrets: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("pre-commit must NOT scan committed-but-unstaged files, got %d findings", len(got))
	}
}
