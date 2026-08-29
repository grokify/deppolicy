package policy

import (
	"path"
	"strings"
)

// InScope reports whether locator falls within the governed ecosystem: it
// must match at least one Scope.Include pattern and no Scope.Exclude
// pattern.
func (p Policy) InScope(locator string) bool {
	included := false
	for _, pattern := range p.Scope.Include {
		if matchLocatorPattern(pattern, locator) {
			included = true
			break
		}
	}
	if !included {
		return false
	}
	for _, pattern := range p.Scope.Exclude {
		if matchLocatorPattern(pattern, locator) {
			return false
		}
	}
	return true
}

// matchLocatorPattern reports whether pattern matches locator.
//
// A pattern ending in "/*" matches the exact prefix before "/*" and
// anything nested beneath it (any depth), which is what lets an org-level
// pattern such as "go:github.com/grokify/*" cover every component,
// including proto-modules several path segments deep.
//
// Any other pattern is matched with path.Match semantics: "*" matches any
// sequence of characters that does not include "/", so a pattern like
// "go:github.com/*/*-archive" matches exactly one org segment and a repo
// segment ending in "-archive", without also matching deeper subpaths.
func matchLocatorPattern(pattern, locator string) bool {
	if prefix, ok := strings.CutSuffix(pattern, "/*"); ok {
		if locator == prefix || strings.HasPrefix(locator, prefix+"/") {
			return true
		}
	}
	matched, err := path.Match(pattern, locator)
	return err == nil && matched
}
