package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/jmcampanini/cubby/internal/cli"
)

func main() {
	cmd := cli.NewRootCommand(os.Stdout, os.Stderr)
	if err := cmd.Execute(); err != nil {
		var exitErr *cli.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
