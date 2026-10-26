package cli

import (
	"errors"
	"os"

	"github.com/charmbracelet/x/term"
)

// requireInteractiveTerminal checks every capability lazygit needs before it
// starts. Tests replace it the way they replace runExternalCommand.
var requireInteractiveTerminal = productionInteractiveTerminal

// productionInteractiveTerminal checks the stream lazygit reads, the stream
// it draws on, and the cursor control TERM advertises, each on its own.
func productionInteractiveTerminal() error {
	if !term.IsTerminal(os.Stdin.Fd()) {
		return errors.New("lazygit needs a terminal on stdin; run cubby lazygit from an interactive shell, not from a pipe")
	}
	if !term.IsTerminal(os.Stdout.Fd()) {
		return errors.New("lazygit needs a terminal on stdout; run cubby lazygit from an interactive shell, not with stdout redirected")
	}
	switch os.Getenv("TERM") {
	case "":
		return errors.New("lazygit needs cursor control but TERM is unset; set TERM to the terminal type, for example xterm-256color")
	case "dumb":
		return errors.New("lazygit needs cursor control, which TERM=dumb does not provide; set TERM to the terminal type, for example xterm-256color")
	}
	return nil
}
