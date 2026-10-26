package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/x/term"
	"github.com/xo/terminfo"
)

// requireInteractiveTerminal checks every capability lazygit needs before it
// starts. Tests replace it the way they replace runExternalCommand.
var requireInteractiveTerminal = productionInteractiveTerminal

// productionInteractiveTerminal checks the stream lazygit reads, the stream
// it draws on, and cursor addressing in the terminfo entry TERM names, each
// on its own.
func productionInteractiveTerminal() error {
	if !term.IsTerminal(os.Stdin.Fd()) {
		return errors.New("lazygit needs a terminal on stdin; run cubby lazygit from an interactive shell, not from a pipe")
	}
	if !term.IsTerminal(os.Stdout.Fd()) {
		return errors.New("lazygit needs a terminal on stdout; run cubby lazygit from an interactive shell, not with stdout redirected")
	}
	termName := os.Getenv("TERM")
	if termName == "" {
		return errors.New("lazygit needs cursor control but TERM is unset; set TERM to the terminal type, for example xterm-256color")
	}
	entry, err := terminfo.LoadFromEnv()
	if err != nil {
		return fmt.Errorf("lazygit needs cursor control but TERM=%s has no terminfo entry (%w); set TERM to the terminal type, for example xterm-256color", termName, err)
	}
	if len(entry.Strings[terminfo.CursorAddress]) == 0 {
		return fmt.Errorf("lazygit needs cursor control, which TERM=%s does not provide; set TERM to the terminal type, for example xterm-256color", termName)
	}
	return nil
}
