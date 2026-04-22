package detect

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// StagedFiles returns files in the git index that are Added/Copied/Modified/Renamed.
// NUL-delimited to survive weird paths.
func StagedFiles(ctx context.Context, repoRoot string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--name-only", "--diff-filter=ACMR", "-z")
	cmd.Dir = repoRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git diff --cached: %w: %s", err, out.String())
	}
	raw := strings.TrimRight(out.String(), "\x00")
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, "\x00")
	return parts, nil
}
