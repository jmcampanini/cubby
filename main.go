// Package main runs Cubby's profile-scoped dotfile commands.
package main

import (
	"os"

	"github.com/jmcampanini/cubby/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
