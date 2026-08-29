//go:build tools

// This file exists only to keep `go mod tidy` from removing
// github.com/invopop/jsonschema, which is used exclusively by the
// //go:build ignore generator in this directory and would otherwise be
// invisible to module dependency analysis.
package main

import (
	_ "github.com/invopop/jsonschema"
)
