package cli

import (
	"strings"
	"testing"
)

func TestFormatValidateReport_Valid(t *testing.T) {
	report := FormatValidateReport(ValidateResult{})
	if !strings.Contains(report, "policy is valid") {
		t.Errorf("report missing valid-policy message, got:\n%s", report)
	}
	if !strings.Contains(report, "0 error(s), 0 warning(s)") {
		t.Errorf("report missing zero-count summary, got:\n%s", report)
	}
}

func TestFormatValidateReport_WithIssues(t *testing.T) {
	result := ValidateResult{
		Issues: []ValidateIssue{
			{Severity: SeverityError, Message: "missing required field"},
			{Severity: SeverityWarning, Message: "expired exception"},
		},
	}
	report := FormatValidateReport(result)

	if !strings.Contains(report, "ERROR  missing required field") {
		t.Errorf("report missing ERROR line, got:\n%s", report)
	}
	if !strings.Contains(report, "WARN   expired exception") {
		t.Errorf("report missing WARN line, got:\n%s", report)
	}
	if !strings.Contains(report, "1 error(s), 1 warning(s)") {
		t.Errorf("report missing correct summary, got:\n%s", report)
	}
}
