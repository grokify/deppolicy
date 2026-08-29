package policy

import "testing"

func TestPolicy_InScope(t *testing.T) {
	p := Policy{
		Scope: Scope{
			Include: []string{
				"go:github.com/grokify/*",
				"go:github.com/plexusone/*",
			},
			Exclude: []string{
				"go:github.com/grokify/sandbox-*",
				"go:github.com/*/*-archive",
				"go:github.com/*/experiment-*",
			},
		},
	}

	tests := []struct {
		name    string
		locator string
		want    bool
	}{
		{"included org, module root", "go:github.com/grokify/mogo", true},
		{"included org, nested proto-module", "go:github.com/grokify/mogo/database", true},
		{"included org, deeply nested", "go:github.com/plexusone/omnillm/providers/openai", true},
		{"different included org", "go:github.com/plexusone/omniagent", true},
		{"out of scope org", "go:github.com/ProductBuildersHQ/prism", false},
		{"third-party", "go:github.com/google/uuid", false},
		{"excluded by sandbox pattern", "go:github.com/grokify/sandbox-foo", false},
		{"exclude pattern without trailing /* does not exclude subpaths", "go:github.com/grokify/sandbox-foo/bar", true},
		{"excluded by archive pattern", "go:github.com/plexusone/legacy-archive", false},
		{"excluded by archive pattern does not exclude subpath", "go:github.com/plexusone/legacy-archive/sub", true},
		{"excluded by experiment pattern", "go:github.com/grokify/experiment-foo", false},
		{"similar name is not falsely excluded", "go:github.com/grokify/sandboxes", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.InScope(tt.locator); got != tt.want {
				t.Errorf("InScope(%q) = %v, want %v", tt.locator, got, tt.want)
			}
		})
	}
}

func TestMatchLocatorPattern_PrefixCollision(t *testing.T) {
	// "*-archive" must not match "legacy-archived" (extra trailing chars).
	if matchLocatorPattern("go:github.com/*/*-archive", "go:github.com/plexusone/legacy-archived") {
		t.Error("pattern falsely matched a locator with extra trailing characters")
	}
}
