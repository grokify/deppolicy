package policy

import (
	"encoding/json"
	"fmt"
	"os"
)

// ParseFile reads, decodes, and structurally validates the policy document
// at path.
func ParseFile(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("policy: read %s: %w", path, err)
	}
	p, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("policy: %s: %w", path, err)
	}
	return p, nil
}

// Parse decodes and structurally validates a policy document.
func Parse(data []byte) (*Policy, error) {
	var p Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("policy: parse: %w", err)
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

// MustParse is like Parse but panics on error. It is intended for embedded
// policy documents that are validated in the source repository's own CI, so
// a parse failure at load time indicates a build/release defect.
func MustParse(data []byte) *Policy {
	p, err := Parse(data)
	if err != nil {
		panic(fmt.Sprintf("policy: MustParse: %v", err))
	}
	return p
}
