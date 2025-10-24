package cli

import "fmt"

// ExitError carries an exit code and message suitable for CLI responses.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Err.Error()
}

// exitErrorf formats an ExitError with the provided code.
func exitErrorf(code int, format string, args ...any) error {
	return &ExitError{Code: code, Err: fmt.Errorf(format, args...)}
}

// exitError wraps err with the supplied exit code.
func exitError(code int, err error) error {
	if err == nil {
		return &ExitError{Code: code}
	}
	return &ExitError{Code: code, Err: err}
}
