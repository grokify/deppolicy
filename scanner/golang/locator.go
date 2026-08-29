// Package golang scans Go modules for dependency-policy evidence: declared
// requirements from go.mod (manifest mode) and, later, direct source
// imports (imports mode). Scanners emit normalized facts; they never make
// policy decisions.
package golang

// Locator returns the policy/graph locator for a Go import path, e.g.
// Locator("github.com/plexusone/omnillm") == "go:github.com/plexusone/omnillm".
func Locator(importPath string) string {
	return "go:" + importPath
}
