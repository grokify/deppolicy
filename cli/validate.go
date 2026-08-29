package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
	"github.com/grokify/deppolicy/schema"
)

// ValidateSeverity classifies a ValidateIssue.
type ValidateSeverity string

const (
	// SeverityError: the policy document is invalid; it should not be
	// used to gate CI.
	SeverityError ValidateSeverity = "error"

	// SeverityWarning: the policy document is valid but has a problem
	// worth a human's attention (an expired exception, a relationship
	// endpoint absent from the supplied graph).
	SeverityWarning ValidateSeverity = "warning"
)

// ValidateIssue is one problem found while validating a policy document.
type ValidateIssue struct {
	Severity ValidateSeverity `json:"severity"`
	Message  string           `json:"message"`
}

// ValidateOptions configures Validate.
type ValidateOptions struct {
	// PolicyPath is the path to a policy.json document. Required.
	PolicyPath string

	// GraphPath, if set, is a previously scanned graph.json used for
	// referential-integrity checks: whether relationship/exception
	// endpoints actually appear as components in that graph. Omit to
	// skip this check.
	GraphPath string

	// AsOf is the evaluation time used to detect expired exceptions.
	AsOf time.Time
}

// ValidateResult is Validate's structured output.
type ValidateResult struct {
	// Policy is non-nil only when the document parsed successfully as a
	// well-formed policy (schema-level issues can still be present).
	Policy *policy.Policy
	Issues []ValidateIssue
}

// OK reports whether result has no SeverityError issues. Warnings alone
// do not make a policy document invalid.
func (r ValidateResult) OK() bool {
	for _, i := range r.Issues {
		if i.Severity == SeverityError {
			return false
		}
	}
	return true
}

// Validate checks a policy document at three layers: JSON Schema
// validation against the embedded policy schema (catching problems Go's
// permissive-by-default JSON unmarshaling would silently miss, such as an
// unrecognized field), structural validation via policy.Parse, and -- if
// opts.GraphPath is set -- referential integrity against a graph, plus
// expired-exception detection. All issues found are collected and
// returned together rather than stopping at the first one, since fixing a
// policy document one issue per run is exactly the friction this command
// exists to avoid.
func Validate(opts ValidateOptions) (ValidateResult, error) {
	data, err := os.ReadFile(opts.PolicyPath)
	if err != nil {
		return ValidateResult{}, fmt.Errorf("cli: read policy %s: %w", opts.PolicyPath, err)
	}

	var issues []ValidateIssue

	schemaIssues, err := validateAgainstSchema(schema.PolicyJSON, data)
	if err != nil {
		return ValidateResult{}, err
	}
	issues = append(issues, schemaIssues...)

	p, parseErr := policy.Parse(data)
	if parseErr != nil {
		issues = append(issues, ValidateIssue{Severity: SeverityError, Message: parseErr.Error()})
		return ValidateResult{Issues: issues}, nil
	}

	for _, e := range p.Exceptions {
		if e.IsExpired(opts.AsOf) {
			issues = append(issues, ValidateIssue{
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("exception %s -> %s expired on %s: %s", e.Source, e.Target, e.Expires, e.Reason),
			})
		}
	}

	if opts.GraphPath != "" {
		graphIssues, err := referentialIntegrityIssues(*p, opts.GraphPath)
		if err != nil {
			return ValidateResult{}, err
		}
		issues = append(issues, graphIssues...)
	}

	return ValidateResult{Policy: p, Issues: issues}, nil
}

func validateAgainstSchema(schemaJSON, data []byte) ([]ValidateIssue, error) {
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("cli: parse embedded policy schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("policy.schema.json", schemaDoc); err != nil {
		return nil, fmt.Errorf("cli: load policy schema: %w", err)
	}
	sch, err := compiler.Compile("policy.schema.json")
	if err != nil {
		return nil, fmt.Errorf("cli: compile policy schema: %w", err)
	}

	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return []ValidateIssue{{Severity: SeverityError, Message: fmt.Sprintf("invalid JSON: %v", err)}}, nil
	}

	if err := sch.Validate(instance); err != nil {
		var valErr *jsonschema.ValidationError
		if errors.As(err, &valErr) {
			return flattenSchemaErrors(valErr), nil
		}
		return []ValidateIssue{{Severity: SeverityError, Message: err.Error()}}, nil
	}

	return nil, nil
}

// flattenSchemaErrors walks a ValidationError's cause tree down to its
// leaves, producing one issue per leaf. A leaf's own Error() string
// already includes its instance location, so no further formatting is
// needed.
func flattenSchemaErrors(err *jsonschema.ValidationError) []ValidateIssue {
	if len(err.Causes) == 0 {
		return []ValidateIssue{{Severity: SeverityError, Message: err.Error()}}
	}
	var issues []ValidateIssue
	for _, cause := range err.Causes {
		issues = append(issues, flattenSchemaErrors(cause)...)
	}
	return issues
}

func referentialIntegrityIssues(p policy.Policy, graphPath string) ([]ValidateIssue, error) {
	data, err := os.ReadFile(graphPath)
	if err != nil {
		return nil, fmt.Errorf("cli: read graph %s: %w", graphPath, err)
	}
	g, err := graph.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("cli: parse graph %s: %w", graphPath, err)
	}

	known := make(map[string]bool, len(g.Components)+len(g.Dependencies)*2)
	for _, c := range g.Components {
		known[c.ID] = true
	}
	for _, d := range g.Dependencies {
		known[d.Source] = true
		known[d.Target] = true
	}

	var issues []ValidateIssue
	checkEndpoint := func(kind string, index int, field, locator string) {
		if !known[locator] {
			issues = append(issues, ValidateIssue{
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("%s[%d]: %s %q does not appear in the supplied graph", kind, index, field, locator),
			})
		}
	}
	for i, r := range p.Relationships {
		checkEndpoint("relationships", i, "source", r.Source)
		checkEndpoint("relationships", i, "target", r.Target)
	}
	for i, e := range p.Exceptions {
		checkEndpoint("exceptions", i, "source", e.Source)
		checkEndpoint("exceptions", i, "target", e.Target)
	}

	return issues, nil
}
