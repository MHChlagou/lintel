package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aegis-sec/aegis/internal/checker"
	"github.com/aegis-sec/aegis/internal/config"
	"github.com/aegis-sec/aegis/internal/detect"
	"github.com/aegis-sec/aegis/internal/finding"
)

type fakeChecker struct {
	name     string
	findings []finding.Finding
	delay    time.Duration
	err      error
}

func (f fakeChecker) Name() string                         { return f.name }
func (fakeChecker) Applicable(*detect.ProjectContext) bool { return true }
func (fakeChecker) RequiredBinaries() []string             { return nil }
func (f fakeChecker) Run(ctx context.Context, _ checker.CheckInput) (checker.CheckOutput, error) {
	select {
	case <-time.After(f.delay):
	case <-ctx.Done():
		return checker.CheckOutput{}, ctx.Err()
	}
	if f.err != nil {
		return checker.CheckOutput{}, f.err
	}
	return checker.CheckOutput{Findings: f.findings}, nil
}

func newSpec() *config.Spec {
	return &config.Spec{
		Performance: config.Performance{
			Parallel:            2,
			CheckTimeoutSeconds: 5,
			TotalTimeoutSeconds: 30,
		},
	}
}

func TestRunnerParallelCollectsAll(t *testing.T) {
	reg := map[string]checker.Checker{
		"a": fakeChecker{name: "a", findings: []finding.Finding{{RuleID: "ra"}}, delay: 10 * time.Millisecond},
		"b": fakeChecker{name: "b", findings: []finding.Finding{{RuleID: "rb"}}, delay: 10 * time.Millisecond},
	}
	mkIn := func(string) checker.CheckInput { return checker.CheckInput{} }
	results := Run(context.Background(), []string{"a", "b"}, mkIn, reg, Options{Spec: newSpec()})
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Err != nil {
			t.Fatalf("%s: %v", r.Name, r.Err)
		}
	}
}

func TestRunnerPropagatesError(t *testing.T) {
	want := errors.New("boom")
	reg := map[string]checker.Checker{
		"x": fakeChecker{name: "x", err: want},
	}
	mkIn := func(string) checker.CheckInput { return checker.CheckInput{} }
	results := Run(context.Background(), []string{"x"}, mkIn, reg, Options{Spec: newSpec()})
	if results[0].Err == nil {
		t.Fatal("expected error")
	}
}

func TestRunnerUnknownCheckFails(t *testing.T) {
	reg := map[string]checker.Checker{}
	mkIn := func(string) checker.CheckInput { return checker.CheckInput{} }
	results := Run(context.Background(), []string{"nope"}, mkIn, reg, Options{Spec: newSpec()})
	if results[0].Err == nil {
		t.Fatal("expected unknown-check error")
	}
}
