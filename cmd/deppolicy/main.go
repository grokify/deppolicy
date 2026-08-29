// Command deppolicy is the CLI for checking Go modules against a central
// dependency policy. It is a thin adapter over the cli package: flag
// parsing and process exit codes live here; everything else is reusable
// and independently testable.
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
