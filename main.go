// Command gegenprobe is the harness. Everything an operator installs is this one
// binary, which is the point of the language decision in
// docs/decisions/0001-the-harness-is-written-in-go.md.
//
// This file is deliberately thin. It reads no arguments, holds no defaults and
// makes no decisions; it hands os.Args to internal/cli and turns the result into
// an exit status. The layout the rest of the tree adds to is:
//
//	main.go            the entry point, and nothing that a test would have to
//	                   build a binary to reach
//	internal/<concern> one package per concern, each nameable in a word: the
//	                   case loader, the runner, a reader, the comparison, the
//	                   report. A package is added when a concern arrives, not
//	                   ahead of it.
//	tools/<name>       commands that are run against the tree rather than
//	                   shipped in the binary
//
// The rule that keeps this file thin is that logic lives where it can be tested
// without a subprocess. Anything reachable only by running the binary is
// something the suite has to build to look at, and a suite that builds is a
// suite people stop running.
package main

import (
	"os"

	"github.com/iderex/gegenprobe/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
