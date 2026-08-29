package golang

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/grokify/deppolicy/component"
	"github.com/grokify/deppolicy/graph"
)

// skipDirNames are directories that never contain source worth attributing
// to architecture: dependency copies, fixtures, and VCS/tooling metadata.
var skipDirNames = map[string]bool{
	"vendor":       true,
	"testdata":     true,
	"node_modules": true,
	".git":         true,
}

// ImportDependencies walks non-test Go source files under dir -- the root
// of a single module -- and returns the module's own locator plus one
// source-import-evidence graph.Dependency per external component its
// packages import. This is V1's authoritative evidence for architecture
// decisions (see graph.EvidenceKindSourceImport); manifest evidence
// (ManifestDependencies) is the fast, coarser fleet-scan counterpart.
//
// Standard library imports and imports of the module's own packages are
// not cross-component dependency edges and are excluded. Only imports
// discovered via non-test files are evidenced: test-only dependencies
// exercise integration surface rather than architectural coupling.
//
// registry, if non-nil, resolves an imported path to its owning governed
// component via longest-declared-prefix match, so a deep import such as
// "github.com/plexusone/omniagent/client" correctly attributes to the
// omniagent module rather than being treated as its own component. Without
// a registry entry covering an import, the import path itself is used
// verbatim as the locator -- correct for module roots, but unverified for
// deeper paths, since only go/packages module resolution could confirm the
// true module boundary of a path this scanner does not govern.
//
// Parsing uses parser.ImportsOnly, so no type checking or full AST parse
// occurs: cost scales with file count and header size, not with source
// size.
func ImportDependencies(dir string, registry *component.Registry) (moduleLocator string, deps []graph.Dependency, err error) {
	moduleLocator, _, err = ManifestDependencies(dir)
	if err != nil {
		return "", nil, err
	}
	modulePath := strings.TrimPrefix(moduleLocator, "go:")

	var raw []graph.Dependency
	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path != dir && (skipDirNames[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileDeps, ferr := fileImportDependencies(dir, path, moduleLocator, modulePath, registry)
		if ferr != nil {
			return ferr
		}
		raw = append(raw, fileDeps...)
		return nil
	})
	if walkErr != nil {
		return "", nil, walkErr
	}

	return moduleLocator, graph.MergeDependencies(raw), nil
}

func fileImportDependencies(dir, path, moduleLocator, modulePath string, registry *component.Registry) ([]graph.Dependency, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("golang: parse %s: %w", path, err)
	}

	relPath, err := filepath.Rel(dir, path)
	if err != nil {
		relPath = path
	}

	var deps []graph.Dependency
	for _, imp := range file.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue // malformed import literal; nothing to record
		}
		if IsStdlib(importPath) {
			continue
		}
		if importPath == modulePath || strings.HasPrefix(importPath, modulePath+"/") {
			continue // intra-module import, not a cross-component dependency
		}

		target := Locator(importPath)
		if registry != nil {
			if c, ok := registry.Resolve(target); ok {
				target = c.Locator
			}
		}

		deps = append(deps, graph.Dependency{
			Source:   moduleLocator,
			Relation: graph.RelationDependsOn,
			Target:   target,
			Evidence: []graph.Evidence{
				{
					Kind: graph.EvidenceKindSourceImport,
					File: relPath,
					Line: fset.Position(imp.Pos()).Line,
				},
			},
		})
	}
	return deps, nil
}

// IsStdlib reports whether importPath looks like a standard library import:
// its first path segment contains no ".", the same heuristic the go
// command's own tooling relies on to distinguish stdlib from module paths
// (a real module domain always contains a dot, e.g. "github.com"). Exported
// for reuse by the analyzer, which classifies imports the same way.
func IsStdlib(importPath string) bool {
	first, _, _ := strings.Cut(importPath, "/")
	return !strings.Contains(first, ".")
}
