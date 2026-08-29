package graph

import (
	"encoding/json"
	"fmt"
)

// Parse decodes and structurally validates a graph document.
func Parse(data []byte) (*Graph, error) {
	var g Graph
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("graph: parse: %w", err)
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return &g, nil
}
