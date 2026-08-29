package graph

import (
	"encoding/json"
	"fmt"
	"os"
)

// WriteFile marshals g as indented JSON and writes it to path.
func (g Graph) WriteFile(path string) error {
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return fmt.Errorf("graph: marshal: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("graph: write %s: %w", path, err)
	}
	return nil
}
