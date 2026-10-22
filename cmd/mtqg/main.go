// Command mtqg records memos, todos, questions and glossary terms in a git
// repository. It only hands its arguments to internal/cli.
package main

import (
	"os"

	"github.com/amisonnet8/mtqg/internal/cli"
)

// version is set by the release build (-ldflags "-X main.version=v0.1.0",
// .goreleaser.yaml). go install leaves it empty.
var version string

func main() {
	env := cli.OSEnv()
	env.Version = version
	os.Exit(cli.Run(env, os.Args[1:]))
}
