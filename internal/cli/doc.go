// Package cli is the command line of mtqg: it reads the arguments, calls the
// core, and shows the result.
//
// It is the only place where mtqg speaks: every sentence it prints is English
// and lives in messages.go, and the core below it returns kinds of error and
// structured results instead (.claude/rules/cli-output.md). Run takes everything
// it touches in an Env (the streams, the environment variables, the clock, the
// terminal), so that tests call it directly.
package cli
