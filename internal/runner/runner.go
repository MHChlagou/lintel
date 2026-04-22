// Package runner executes checkers concurrently under timeout and parallelism bounds.
package runner

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/aegis-sec/aegis/internal/checker"
	"github.com/aegis-sec/aegis/internal/config"
)

type Result struct {
	Name   string
	Output checker.CheckOutput
	Err    error
}

type Options struct {
	Spec     *config.Spec
	FailFast bool
}

// Run executes the given checks in parallel, bounded by spec.performance.parallel.
// Each check runs with its own timeout; the whole run is bounded by the total timeout.
// Results preserve input order on exit.
func Run(parent context.Context, checks []string, mkInput func(name string) checker.CheckInput, reg map[string]checker.Checker, opts Options) []Result {
	parallel := resolveParallel(opts.Spec.Performance.Parallel)
	totalTO := time.Duration(opts.Spec.Performance.TotalTimeoutSeconds) * time.Second
	checkTO := time.Duration(opts.Spec.Performance.CheckTimeoutSeconds) * time.Second

	ctx, cancel := context.WithTimeout(parent, totalTO)
	defer cancel()

	sem := make(chan struct{}, parallel)
	results := make([]Result, len(checks))
	var wg sync.WaitGroup

	// failFastCancel fires when the first blocking finding appears and opts.FailFast.
	failFastCtx, failFastCancel := context.WithCancel(ctx)
	defer failFastCancel()

	for i, name := range checks {
		i, name := i, name
		chk, ok := reg[name]
		if !ok {
			results[i] = Result{Name: name, Err: fmt.Errorf("unknown check %q", name)}
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-failFastCtx.Done():
				results[i] = Result{Name: name, Err: failFastCtx.Err()}
				return
			}
			defer func() { <-sem }()

			runCtx, runCancel := context.WithTimeout(failFastCtx, checkTO)
			defer runCancel()

			out, err := chk.Run(runCtx, mkInput(name))
			results[i] = Result{Name: name, Output: out, Err: err}

			if opts.FailFast && err == nil {
				for _, f := range out.Findings {
					if f.Blocking { // gate not yet applied; this is conservative
						failFastCancel()
						break
					}
				}
			}
		}()
	}
	wg.Wait()
	return results
}

func resolveParallel(v any) int {
	switch x := v.(type) {
	case int:
		if x > 0 {
			return x
		}
	case float64:
		if int(x) > 0 {
			return int(x)
		}
	case string:
		if x == "auto" || x == "" {
			return runtime.NumCPU()
		}
	}
	return runtime.NumCPU()
}
