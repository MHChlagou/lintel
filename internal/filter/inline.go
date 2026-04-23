package filter

import (
	"bufio"
	"os"
	"strings"
)

// InlineIgnored reports whether the given file:line (or the preceding line)
// carries an "lintel:ignore-..." directive with a non-empty reason.
//
// Accepted forms (on the same or preceding line):
//
//	// lintel:ignore-secret  reason="test fixture"
//	# lintel:ignore-rule=SQLi.raw-concat  reason="hardened elsewhere"
//
// A bare `lintel:ignore-*` without a reason is itself a finding caller can surface.
func InlineIgnored(path string, line int, marker, rule string) (bool, bool, error) {
	if line <= 0 {
		return false, false, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return false, false, err
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	current := 0
	var prev, cur string
	for scanner.Scan() {
		current++
		prev = cur
		cur = scanner.Text()
		if current == line {
			break
		}
	}
	for _, candidate := range []string{cur, prev} {
		if !strings.Contains(candidate, marker) && !strings.Contains(candidate, "lintel:ignore") {
			continue
		}
		// Rule-scoped ignores must name the rule.
		if strings.Contains(candidate, "lintel:ignore-rule=") {
			want := "lintel:ignore-rule=" + rule
			if !strings.Contains(candidate, want) {
				continue
			}
		}
		hasReason := strings.Contains(candidate, "reason=")
		return true, !hasReason, nil
	}
	return false, false, nil
}
