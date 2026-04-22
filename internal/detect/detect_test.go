package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func makeRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for path, body := range files {
		p := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestDetectExplicit(t *testing.T) {
	ctx, err := Detect("/nowhere", []string{"go", "npm"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.Stacks) != 2 {
		t.Fatalf("want 2 stacks, got %v", ctx.Stacks)
	}
}

func TestDetectManifestNpmVariants(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"npm", map[string]string{"package.json": "{}", "package-lock.json": "{}"}, Npm},
		{"pnpm", map[string]string{"package.json": "{}", "pnpm-lock.yaml": ""}, Pnpm},
		{"yarn", map[string]string{"package.json": "{}", "yarn.lock": ""}, Yarn},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := makeRepo(t, c.files)
			ctx, err := Detect(root, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(ctx.Stacks) != 1 || ctx.Stacks[0] != c.want {
				t.Fatalf("want [%s], got %v", c.want, ctx.Stacks)
			}
		})
	}
}

func TestDetectGoAndPython(t *testing.T) {
	root := makeRepo(t, map[string]string{
		"go.mod":           "module x",
		"requirements.txt": "",
	})
	ctx, err := Detect(root, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.Stacks) != 2 {
		t.Fatalf("want 2 stacks, got %v", ctx.Stacks)
	}
}

func TestExtensionFallback(t *testing.T) {
	root := makeRepo(t, map[string]string{"nothing.txt": ""})
	staged := []string{"a.go", "b.go", "c.py"}
	ctx, err := Detect(root, nil, nil, staged)
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.Stacks) != 1 || ctx.Stacks[0] != Go {
		t.Fatalf("want [go], got %v", ctx.Stacks)
	}
}
