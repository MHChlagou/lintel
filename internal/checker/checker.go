// Package checker defines the plugin-shaped interface every built-in scanner adapter implements.
package checker

import (
	"context"
	"encoding/json"

	"github.com/MHChlagou/lintel/internal/config"
	"github.com/MHChlagou/lintel/internal/detect"
	"github.com/MHChlagou/lintel/internal/finding"
	"github.com/MHChlagou/lintel/internal/resolve"
)

type Stats struct {
	FilesScanned int  `json:"files_scanned"`
	DurationMS   int  `json:"duration_ms"`
	TimedOut     bool `json:"timed_out"`
}

type CheckInput struct {
	RepoRoot    string
	StagedFiles []string
	FullTree    bool
	Config      json.RawMessage
	Spec        *config.Spec
	Project     *detect.ProjectContext
	Resolver    *resolve.Resolver
	Hook        string // "pre-commit" | "pre-push" | ""
}

type CheckOutput struct {
	Findings []finding.Finding
	Stats    Stats
}

type Checker interface {
	Name() string
	Applicable(ctx *detect.ProjectContext) bool
	Run(ctx context.Context, in CheckInput) (CheckOutput, error)
	RequiredBinaries() []string
}
