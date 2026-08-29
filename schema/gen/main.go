//go:build ignore

// Command gen generates JSON Schema documents from the policy.Policy and
// graph.Graph Go types, which are the source of truth. Run via
// `go generate ./schema/...` from the repository root.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/invopop/jsonschema"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if err := generate(&policy.Policy{}, "policy.schema.json"); err != nil {
		return err
	}
	if err := generate(&graph.Graph{}, "graph.schema.json"); err != nil {
		return err
	}
	return nil
}

func generate(v any, filename string) error {
	reflector := &jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(v)
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal schema for %s: %w", filename, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filename, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", filename, err)
	}
	return nil
}
