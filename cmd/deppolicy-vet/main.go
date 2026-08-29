// Command deppolicy-vet runs the deppolicy analyzer as a standalone
// vet-style tool:
//
//	deppolicy-vet -policy=policy.json ./...
//	go vet -vettool=$(which deppolicy-vet) -policy=policy.json ./...
//
// Violations are reported at the offending import line.
package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/grokify/deppolicy/analyzer"
)

func main() {
	singlechecker.Main(analyzer.Analyzer)
}
