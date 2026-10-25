// Package main runs Cubby's profile-scoped dotfile commands.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/jmcampanini/cubby/internal/cli"
)

func main() {
	if err := cli.Execute(os.Stdout, os.Stderr); err != nil {
		var exitErr *cli.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		_, _ = fmt.Fprintf(os.Stderr, "cubby: error: %v\n", err)
		os.Exit(1)
	}
}
