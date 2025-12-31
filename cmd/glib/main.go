// Glib CLI - Code generation framework for Go web applications
package main

import (
	"fmt"
	"os"

	"github.com/azizndao/glib/internal/cli"
)

var version = "0.1.0-dev"

func main() {
	if err := cli.Execute(version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
