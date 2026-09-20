// Package mtqg holds the format specification that mtqg embeds into its binary.
//
// Go can only embed files below the package directory, so this package sits at
// the repository root, next to docs/reference/schema.md. It has no logic.
package mtqg

import _ "embed"

// Schema is docs/reference/schema.md. mtqg init writes it to .mtqg/SCHEMA.md.
//
//go:embed docs/reference/schema.md
var Schema string
