// Package consumer is the analysistest subject: its module identity is
// supplied via the -deppolicy.module flag override (analysistest's GOPATH
// driver provides no module info).
package consumer

import (
	"fmt"

	"github.com/orgA/badtarget" // want `may not depend on go:github\.com/orgA/badtarget`
	"github.com/orgA/foundlib"
	"github.com/orgA/provider"
)

func Use() {
	fmt.Println("ok")
	provider.Do()
	foundlib.Do()
	badtarget.Do()
}
