package graph

import "fmt"

// Validate checks that the graph document is well-formed. It does not check
// the edges against a policy -- that comparison is the evaluator's job.
func (g Graph) Validate() error {
	if g.Scan.Level != ScanLevelManifest && g.Scan.Level != ScanLevelSourceImports {
		return fmt.Errorf("graph: scan.level %q is not a recognized scan level", g.Scan.Level)
	}
	if g.Scan.Scanner == "" {
		return fmt.Errorf("graph: scan.scanner is required")
	}

	for i, d := range g.Dependencies {
		if d.Source == "" || d.Target == "" {
			return fmt.Errorf("graph: dependencies[%d]: source and target are required", i)
		}
		if d.Relation != RelationDependsOn {
			return fmt.Errorf("graph: dependencies[%d]: unsupported relation %q (V1 supports only %q)", i, d.Relation, RelationDependsOn)
		}
		for j, e := range d.Evidence {
			if !validEvidenceKind(e.Kind) {
				return fmt.Errorf("graph: dependencies[%d].evidence[%d]: unrecognized kind %q", i, j, e.Kind)
			}
		}
	}

	return nil
}

func validEvidenceKind(k EvidenceKind) bool {
	switch k {
	case EvidenceKindManifest, EvidenceKindSourceImport, EvidenceKindIaC, EvidenceKindLLMAnalysis, EvidenceKindTelemetry:
		return true
	default:
		return false
	}
}
