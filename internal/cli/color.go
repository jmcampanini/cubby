package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

const colorFlagName = "color"

// colorMode is the --color value: when cubby styles the output it renders.
type colorMode string

const (
	colorAuto   colorMode = "auto"
	colorAlways colorMode = "always"
	colorNever  colorMode = "never"
)

func (m *colorMode) String() string { return string(*m) }

// Set rejects any value outside the fixed set, naming it, so the rejection
// happens where the flag is parsed.
func (m *colorMode) Set(value string) error {
	switch colorMode(value) {
	case colorAuto, colorAlways, colorNever:
		*m = colorMode(value)
		return nil
	default:
		return fmt.Errorf("unknown color mode %q; use auto, always, or never", value)
	}
}

func (m *colorMode) Type() string { return "mode" }

func addColorFlag(cmd *cobra.Command) {
	mode := colorAuto
	cmd.PersistentFlags().Var(&mode, colorFlagName, "style output: auto, always, or never")
}

// styledOutput reports whether cmd may style what it writes to out under the
// --color mode in effect.
func styledOutput(cmd *cobra.Command, out io.Writer) bool {
	switch colorMode(cmd.Flag(colorFlagName).Value.String()) {
	case colorAlways:
		return true
	case colorNever:
		return false
	default:
		if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
			return false
		}
		file, ok := out.(interface{ Fd() uintptr })
		return ok && term.IsTerminal(file.Fd())
	}
}
