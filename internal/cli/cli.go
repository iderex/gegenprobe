// Package cli turns the arguments a command line supplies into a call and an
// exit status. It is the only place that decides what a subcommand is called
// and what it writes, so the command at the root of the tree holds no logic and
// the behaviour below is testable without starting a process.
package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/iderex/gegenprobe/internal/version"
)

// name is what the program calls itself in its own messages. It is fixed rather
// than taken from argv[0], so that a renamed or relinked binary reports the
// same name a reader can search for.
const name = "gegenprobe"

// usage is the whole of what this command can do today. There are two
// subcommands and neither of them runs a case yet, which the text says plainly:
// a usage message that describes an intention rather than a binary sends an
// operator looking for a flag that does not exist.
const usage = `gegenprobe compares what several atomic structure codes say about the same case.

usage:
    gegenprobe <subcommand>

subcommands:
    version    print the version of this binary
    help       print this text

Nothing here runs a case yet. This binary is the skeleton the rest is added to,
and it is honest about being one.
`

// Run executes one invocation and returns the exit status. Everything it writes
// goes to one of the two writers it is given rather than to the process
// streams, so a test reads the output instead of capturing a process.
//
// The status is zero where the command did what was asked, and two where the
// arguments were not something this command can do. Two rather than one, so
// that a caller can tell a usage mistake from a failure of the work, on the day
// there is work to fail.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintf(stderr, "%s: no subcommand given.\n\n", name)
		fmt.Fprint(stderr, usage)
		return 2
	}

	switch args[0] {
	case "version":
		if len(args) > 1 {
			fmt.Fprintf(stderr, "%s: version takes no arguments, and got: %s\n", name, strings.Join(args[1:], " "))
			return 2
		}
		fmt.Fprintln(stdout, version.Resolve())
		return 0

	case "help":
		if len(args) > 1 {
			fmt.Fprintf(stderr, "%s: help takes no arguments, and got: %s\n", name, strings.Join(args[1:], " "))
			return 2
		}
		fmt.Fprint(stdout, usage)
		return 0

	default:
		fmt.Fprintf(stderr, "%s: %q is not a subcommand of this binary.\n\n", name, args[0])
		fmt.Fprint(stderr, usage)
		return 2
	}
}
