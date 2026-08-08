// Package cli is the command's argument handling and nothing else. It writes to
// the writers it is given and returns an exit status rather than calling
// os.Exit, so that every path through it is reachable from a test without
// building a binary or trapping a process.
package cli

import (
	"fmt"
	"io"

	"github.com/iderex/gegenprobe/internal/version"
)

// Name is what the command calls itself in its own usage text.
const Name = "gegenprobe"

// Summary is the one line a reader arriving at the usage text gets first.
const Summary = "A harness that runs the atomic structure codes against each other systematically."

// Exit statuses. Two rather than one for misuse follows the convention a shell
// user already has, where one is a failed job and two is a misused command.
const (
	statusOK     = 0
	statusMisuse = 2
)

type command struct {
	name    string
	summary string
	run     func(stdout io.Writer)
}

// commands is the whole set. It is a function rather than a package variable so
// that no other package can add to it from an init, which is how a command set
// becomes something nobody can enumerate by reading.
func commands() []command {
	return []command{
		{
			name:    "version",
			summary: "print the version of this build",
			run:     func(stdout io.Writer) { fmt.Fprintln(stdout, version.Describe()) },
		},
		{
			name:    "help",
			summary: "print this usage",
			run:     usage,
		},
	}
}

// Run executes one subcommand and returns the exit status. Usage goes to stderr
// when it is the answer to a mistake and to stdout when it is what was asked
// for, so that a reader piping the command somewhere gets what they asked for
// and a script gets its diagnostics out of the way of its data.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintf(stderr, "%s: no subcommand given\n", Name)
		usage(stderr)
		return statusMisuse
	}

	name, rest := args[0], args[1:]
	for _, c := range commands() {
		if c.name != name {
			continue
		}
		if len(rest) != 0 {
			fmt.Fprintf(stderr, "%s: %s takes no arguments, got %d\n", Name, c.name, len(rest))
			usage(stderr)
			return statusMisuse
		}
		c.run(stdout)
		return statusOK
	}

	fmt.Fprintf(stderr, "%s: unknown subcommand %q\n", Name, name)
	usage(stderr)
	return statusMisuse
}

func usage(w io.Writer) {
	fmt.Fprintf(w, "usage: %s <subcommand>\n\n", Name)
	fmt.Fprintf(w, "%s\n\n", Summary)
	fmt.Fprintln(w, "Subcommands:")
	for _, c := range commands() {
		fmt.Fprintf(w, "  %-9s %s\n", c.name, c.summary)
	}
}
