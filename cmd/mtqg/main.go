// Command mtqg records memos, todos, questions and glossary terms in a git
// repository. It only hands its arguments to internal/cli.
package main

import (
	"os"

	"github.com/amisonnet8/mtqg/internal/cli"
)

func main() {
	os.Exit(cli.Run(cli.OSEnv(), os.Args[1:]))
}
