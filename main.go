package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/jmcampanini/cubby/cmd"
)

func main() {
	root := cmd.NewRootCommand(os.Stdout, os.Stderr)
	if err := root.Execute(); err != nil {
		var exitErr *cmd.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
