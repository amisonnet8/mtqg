package cli

import (
	"errors"
	"runtime/debug"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// version is set at build time (-ldflags "-X .../internal/cli.version=v0.1.0").
// Without it, the version comes from the build information.
var version string

// buildVersion says which mtqg this is: the version set at build time, else the
// version of the module in the build information (a tag for go install
// ...@v0.1.0, a version made of the commit time and hash for go build), else
// dev with the commit if it is known.
func buildVersion() string {
	if version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 7 {
			return "dev (" + s.Value[:7] + ")"
		}
	}
	return "dev"
}

// runVersion shows the version of mtqg and, apart from it, the format version
// of the repository. Where there is no repository the second says so, and it is
// not an error.
func runVersion(c *ctx) int {
	c.println(msgVersion(buildVersion()))

	found, known := 0, false
	if start, err := c.startDir(); err == nil {
		j, err := journal.Open(start, journal.Options{})
		var tooNew *journal.FormatTooNewError
		switch {
		case err == nil:
			found, known = j.Version(), true
		case errors.As(err, &tooNew):
			found, known = tooNew.Found, true
		}
	}
	c.println(msgFormatVersion(found, journal.SupportedVersion, known))
	return exitOK
}
