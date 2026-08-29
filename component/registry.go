package component

import (
	"fmt"
	"sort"
	"strings"
)

// Registry resolves ecosystem locators (e.g. observed import paths) to their
// governing Component using longest-declared-prefix matching. A Go module is
// the default component for its import-path prefix; a declared sub-path
// (a proto-module) carves a finer-grained component out of its parent.
type Registry struct {
	byID     map[ID]Component
	byPrefix []Component // sorted longest-locator-first
}

// NewRegistry builds a Registry from a set of components. It returns an
// error if two components share an ID or an identical Locator.
func NewRegistry(components []Component) (*Registry, error) {
	r := &Registry{
		byID: make(map[ID]Component, len(components)),
	}
	seenLocator := make(map[string]ID, len(components))

	for _, c := range components {
		if _, dup := r.byID[c.ID]; dup {
			return nil, fmt.Errorf("component: duplicate component id %q", c.ID)
		}
		if owner, dup := seenLocator[c.Locator]; dup {
			return nil, fmt.Errorf("component: locator %q declared by both %q and %q", c.Locator, owner, c.ID)
		}
		seenLocator[c.Locator] = c.ID
		r.byID[c.ID] = c
		r.byPrefix = append(r.byPrefix, c)
	}

	sort.SliceStable(r.byPrefix, func(i, j int) bool {
		return len(r.byPrefix[i].Locator) > len(r.byPrefix[j].Locator)
	})

	return r, nil
}

// Get returns the component with the given ID.
func (r *Registry) Get(id ID) (Component, bool) {
	c, ok := r.byID[id]
	return c, ok
}

// All returns every registered component. The returned slice is a copy and
// safe for the caller to mutate.
func (r *Registry) All() []Component {
	out := make([]Component, len(r.byPrefix))
	copy(out, r.byPrefix)
	return out
}

// Resolve returns the governing component for the given locator by
// longest-declared-prefix match: the locator either equals a declared
// component locator, or begins with a declared locator followed by "/".
// This prevents "omnillm2" from falsely matching a declared "omnillm"
// prefix. It returns false if no declared component governs the locator.
func (r *Registry) Resolve(locator string) (Component, bool) {
	for _, c := range r.byPrefix {
		if locator == c.Locator || strings.HasPrefix(locator, c.Locator+"/") {
			return c, true
		}
	}
	return Component{}, false
}
