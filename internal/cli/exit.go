package cli

import "fmt"

// Exit codes documented in docs/SPEC.md.
const (
	ExitOK      = 0 // success
	ExitUsage   = 1 // bad flags, validation, usage errors
	ExitRuntime = 2 // I/O, archive, password, and other runtime failures
)

// ExitError carries a process exit code with a user-facing message.
// Messages should be actionable without requiring -v.
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string {
	return e.Message
}

// Usagef builds a validation/usage ExitError (code 1).
func Usagef(format string, args ...any) *ExitError {
	return &ExitError{Code: ExitUsage, Message: fmt.Sprintf(format, args...)}
}

// Runtimef builds a runtime ExitError (code 2).
func Runtimef(format string, args ...any) *ExitError {
	return &ExitError{Code: ExitRuntime, Message: fmt.Sprintf(format, args...)}
}
