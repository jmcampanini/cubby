package cli

// exitError asks Execute to print Message and exit with Code.
type exitError struct {
	Code    int
	Message string
}

func (e *exitError) Error() string {
	return e.Message
}
