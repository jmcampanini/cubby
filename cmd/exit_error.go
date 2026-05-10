package cmd

import "fmt"

// ExitError asks main to exit with Code without printing an additional error message.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}
