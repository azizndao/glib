// Glib CLI - Code generation framework for Go web applications
package main

import (
	"fmt"
	"os"

	"github.com/goyave/glib/v2/internal/cli"
)

var version = "2.0.0-dev"

func main() {
	if err := cli.Execute(version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
