// Package analyzer exposes DepPolicy's default-deny dependency check as a
// go/analysis Analyzer, so violations surface at the offending import line
// in `go vet -vettool`, editors, and any multichecker -- the enforcement
// point closest to where an import gets written.
//
// The analyzer is a PEP (policy enforcement point) over the same pure
// evaluator every other frontend uses: each package's ecosystem imports
// become observed edges and Policy.EvaluateWithStatus decides. It never
// makes its own authorization judgments.
//
// Two deliberate differences from the repository scanner:
//
//   - No baseline. Every observed edge to a deprecated component is
//     treated as new (Policy.EvaluateWithStatus with a nil baseline), so
//     the analyzer is strictly ratchet-tightening at the editor.
//   - Module resolution is heuristic for undeclared targets. The scanner
//     resolves deep import paths against a registry of every discovered
//     module; the analyzer only has the policy in hand, so an import that
//     matches no declared component locator falls back to collapsing
//     "host/org/repo/deep/pkg" to its first three segments -- correct for
//     the overwhelming github.com-style case, wrong only for nested
//     modules, which should be declared as components anyway.
package analyzer

import (
	"flag"
	"fmt"
	"go/ast"
	"strconv"
	"strings"
	"time"

	"golang.org/x/tools/go/analysis"

	"github.com/grokify/deppolicy/component"
	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
	"github.com/grokify/deppolicy/scanner/golang"
)

// Analyzer is the deppolicy go/analysis analyzer. Configure it with:
//
//	-policy <path>   policy.json to enforce (required)
//	-module <path>   module path override (normally taken from the build
//	                 system; needed only under drivers that don't supply
//	                 module info, e.g. analysistest)
//
// Flag names are unprefixed under both the standalone deppolicy-vet binary
// and `go vet -vettool=deppolicy-vet` (verified against both); a
// multichecker embedding this analyzer would expose them as
// -deppolicy.policy / -deppolicy.module per its usual namespacing.
var Analyzer = &analysis.Analyzer{
	Name:  "deppolicy",
	Doc:   "report imports that violate the ecosystem dependency policy (default deny)",
	Flags: flags(),
	Run:   run,
}

func flags() flag.FlagSet {
	var fs flag.FlagSet
	fs.String("policy", "", "path to policy.json (required)")
	fs.String("module", "", "module path override for drivers without module info")
	return fs
}

func run(pass *analysis.Pass) (any, error) {
	policyPath := pass.Analyzer.Flags.Lookup("policy").Value.String()
	if policyPath == "" {
		return nil, fmt.Errorf("deppolicy: the -deppolicy.policy flag is required")
	}
	p, err := policy.ParseFile(policyPath)
	if err != nil {
		return nil, err
	}

	modulePath := pass.Analyzer.Flags.Lookup("module").Value.String()
	if modulePath == "" && pass.Module != nil {
		modulePath = pass.Module.Path
	}
	if modulePath == "" {
		return nil, nil // no module context (e.g. an ad hoc driver); nothing to attribute imports to
	}

	sourceLocator := golang.Locator(modulePath)
	if !p.InScope(sourceLocator) {
		return nil, nil // module outside the governed ecosystem
	}

	registry, err := declaredRegistry(p)
	if err != nil {
		return nil, err
	}

	// Collect this package's distinct ecosystem edges, remembering every
	// import position per target so a violation is reported at each
	// offending line, not just the first.
	positions := make(map[string][]*ast.ImportSpec)
	var deps []graph.Dependency
	for _, file := range pass.Files {
		if isTestFile(pass, file) {
			continue // test-only imports are integration surface, not architecture (matches the scanner)
		}
		for _, imp := range file.Imports {
			importPath, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}
			if golang.IsStdlib(importPath) {
				continue
			}
			if importPath == modulePath || strings.HasPrefix(importPath, modulePath+"/") {
				continue
			}

			target := resolveTarget(importPath, registry)
			if _, seen := positions[target]; !seen {
				deps = append(deps, graph.Dependency{
					Source:   sourceLocator,
					Relation: graph.RelationDependsOn,
					Target:   target,
				})
			}
			positions[target] = append(positions[target], imp)
		}
	}
	if len(deps) == 0 {
		return nil, nil
	}

	g := graph.Graph{
		Scan:         graph.ScanMetadata{Level: graph.ScanLevelSourceImports, Scanner: "deppolicy-analyzer"},
		Dependencies: deps,
	}

	for _, f := range p.EvaluateWithStatus(g, time.Now(), nil) {
		if f.Status != policy.FindingViolation {
			continue
		}
		for _, imp := range positions[f.Target] {
			pass.Reportf(imp.Pos(), "%s may not depend on %s: %s", sourceLocator, f.Target, f.Reason)
		}
	}

	return nil, nil
}

// declaredRegistry builds a locator-resolution registry from the policy's
// component declarations, so deep imports into declared components (e.g. a
// proto-module carve-out, or a module root with subpackages) attribute to
// the declared locator via longest-prefix match.
func declaredRegistry(p *policy.Policy) (*component.Registry, error) {
	components := make([]component.Component, 0, len(p.Components))
	for locator, decl := range p.Components {
		components = append(components, component.Component{
			ID:      component.ID(locator),
			Kind:    decl.Kind,
			Locator: locator,
			Status:  decl.Status,
		})
	}
	return component.NewRegistry(components)
}

// resolveTarget maps an import path to its owning component locator:
// declared components win by longest prefix; otherwise a github.com-style
// path collapses to host/org/repo (see the package comment for why this
// heuristic is acceptable here and not in the scanner).
func resolveTarget(importPath string, registry *component.Registry) string {
	locator := golang.Locator(importPath)
	if c, ok := registry.Resolve(locator); ok {
		return c.Locator
	}
	parts := strings.SplitN(importPath, "/", 4)
	if len(parts) >= 3 {
		return golang.Locator(strings.Join(parts[:3], "/"))
	}
	return locator
}

// isTestFile reports whether file's filename ends in _test.go.
func isTestFile(pass *analysis.Pass, file *ast.File) bool {
	return strings.HasSuffix(pass.Fset.Position(file.Pos()).Filename, "_test.go")
}
