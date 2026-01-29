// Package version provides a single source of truth for the application version.
// Version can be set at build time via ldflags: -ldflags '-X github.com/invisible-tech/autopilot-security-sensor/internal/version.Version=1.2.3'
package version

// Version is set at build time; default for local builds.
var Version = "0.1.0"
