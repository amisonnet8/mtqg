//go:build !windows

package cli

// enableANSI says whether the terminal understands color codes. Terminals of
// these systems do; TERM=dumb is looked at by the caller.
func enableANSI() bool { return true }
