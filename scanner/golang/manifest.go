package golang

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"

	"github.com/grokify/deppolicy/graph"
)

// ManifestDependencies parses the go.mod file in dir and returns the
// module's own locator plus one manifest-evidence Dependency edge per
// direct require statement.
//
// Requires marked "// indirect" are excluded: an indirect requirement means
// no package in this module directly imports it, so it is not a declared
// direct dependency of this module's own code, only of something it
// transitively needs. This is the fast, cheap counterpart to import-mode
// scanning (see ImportDependencies): it needs to read only one small file
// per repository, which is what makes fleet-wide manifest scans over
// hundreds of repositories -- or a single API fetch per repo with no
// cloning at all -- practical.
func ManifestDependencies(dir string) (moduleLocator string, deps []graph.Dependency, err error) {
	path := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("golang: read %s: %w", path, err)
	}
	return parseManifest(path, data)
}

// ParseManifest is like ManifestDependencies but takes go.mod content
// directly, for callers that already have it in hand (e.g. fetched from an
// SCM API without cloning). name is used only for error messages.
func ParseManifest(name string, data []byte) (moduleLocator string, deps []graph.Dependency, err error) {
	return parseManifest(name, data)
}

func parseManifest(name string, data []byte) (moduleLocator string, deps []graph.Dependency, err error) {
	mf, err := modfile.Parse(name, data, nil)
	if err != nil {
		return "", nil, fmt.Errorf("golang: parse %s: %w", name, err)
	}
	if mf.Module == nil {
		return "", nil, fmt.Errorf("golang: %s has no module directive", name)
	}

	moduleLocator = Locator(mf.Module.Mod.Path)

	for _, req := range mf.Require {
		if req.Indirect {
			continue
		}
		deps = append(deps, graph.Dependency{
			Source:   moduleLocator,
			Relation: graph.RelationDependsOn,
			Target:   Locator(req.Mod.Path),
			Evidence: []graph.Evidence{
				{Kind: graph.EvidenceKindManifest, Source: "go.mod"},
			},
		})
	}

	return moduleLocator, deps, nil
}
