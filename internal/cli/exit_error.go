package cli

import "fmt"

// exitError asks Execute to exit with Code without printing an additional error message.
type exitError struct {
	Code int
}

func (e *exitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}
