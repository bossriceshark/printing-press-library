// Copyright 2026 Cursor Agent. Licensed under Apache-2.0. See LICENSE.
package cli

import "errors"

type cliError struct {
	code int
	err  error
}

func (e *cliError) Error() string { return e.err.Error() }
func (e *cliError) Unwrap() error { return e.err }

func codeErr(code int, err error) error {
	if err == nil {
		return nil
	}
	var existing *cliError
	if errors.As(err, &existing) {
		return err
	}
	return &cliError{code: code, err: err}
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var ce *cliError
	if errors.As(err, &ce) {
		return ce.code
	}
	msg := err.Error()
	switch {
	case containsAny(msg, "unknown command", "unknown flag", "accepts ", "required", "invalid argument"):
		return 2
	default:
		return 1
	}
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if len(p) > 0 && len(s) >= len(p) && (s == p || len(s) > 0 && stringContains(s, p)) {
			return true
		}
	}
	return false
}

func stringContains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || len(find(s, sub)) > 0))
}

func find(s, sub string) string {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return s[i:]
		}
	}
	return ""
}
