// Package version exposes build-time metadata. Values are overridden at link time.
package version

const (
	SchemaVersion = 1
)

var (
	Version = "0.1.0-dev"
	Commit  = "none"
	Date    = "unknown"
)
