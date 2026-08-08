// Command gegenprobe runs a case through several atomic structure codes and
// says where they agree.
//
// This file is the whole of the command. It reads the arguments the operating
// system handed over, hands them to one function, and turns what that function
// returns into an exit status. Nothing else belongs here.
//
// That is the package layout rule for this repository, decided here because
// every later issue adds to it. One package under internal/ per concern, named
// for the concern rather than for the layer, and the command at the root stays
// thin enough that there is never a reason to write a test against it that is
// really a test of something else. A concern that grows a second responsibility
// becomes a second package rather than a second file in the first one.
//
// The rule is a convention until something refuses a violation of it. Nothing
// in this tree does today; the repository invariant lint in the quality
// milestone is where an import graph would be judged, and until it lands this
// comment is a description.
package main

import (
	"os"

	"github.com/iderex/gegenprobe/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
