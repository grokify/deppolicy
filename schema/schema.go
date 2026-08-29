// Package schema embeds the JSON Schema documents generated from the
// policy.Policy and graph.Graph Go types. The Go structs are the source of
// truth; these schemas are a generated, embeddable artifact for non-Go
// consumers and for validating hand-authored documents.
package schema

import _ "embed"

//go:generate go run gen/main.go

//go:embed policy.schema.json
var PolicyJSON []byte

//go:embed graph.schema.json
var GraphJSON []byte
