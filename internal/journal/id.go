package journal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// idLen is the length of a full ID: 32 lowercase hex digits.
const idLen = 32

// newID returns a random UUID (version 4) as 32 lowercase hex digits without
// hyphens. IDs need no coordination between clones, branches or machines, so
// nothing but the random source is involved (.claude/rules/journal-format.md).
func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("journal: generate an ID: %w", err)
	}
	b[6] = b[6]&0x0f | 0x40 // version 4
	b[8] = b[8]&0x3f | 0x80 // variant 10
	return hex.EncodeToString(b[:]), nil
}

// isID reports whether s is a full ID: 32 lowercase hex digits. Stored data
// always holds full IDs; short IDs are for display and input only.
func isID(s string) bool {
	if len(s) != idLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
