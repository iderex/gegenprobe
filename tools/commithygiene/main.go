// Command commithygiene judges the commits a change is made of against
// docs/commit-messages.md: that every subject names the issue it belongs to, and
// that every message is written in the characters that document allows.
//
// It is one command with two callers. The leg named commit hygiene in
// "go run ./gate" runs the same judgement over the range between the default
// branch and the work, so a contributor meets the verdict before pushing. This
// command is what the job named Commit hygiene runs over the range a pull
// request actually holds, which is the only place the base and the head are
// known. Neither holds any of the judgement, which lives in internal/commit and
// is where the fixtures are.
//
// The exit status distinguishes three things, because a caller that could not
// tell them apart would report a range it never had as a range it found nothing
// in:
//
//	0  every commit in the range was examined and nothing refuses it
//	1  something refuses it, and what was refused is on stdout
//	2  no range could be determined, and what was missing is on stderr
//	3  the run could not be made at all
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/iderex/gegenprobe/internal/commit"
)

const (
	statusPass    = 0
	statusFail    = 1
	statusNoRange = 2
	statusBroken  = 3
)

func main() {
	doc := flag.String("doc", "docs/commit-messages.md", "the declaration the rules are read from")
	base := flag.String("base", "origin/main", "the ref the change is measured against")
	head := flag.String("head", "HEAD", "the ref the change ends at")
	external := flag.Bool("external", false, "the head is not a branch of this repository, so a subject naming no issue is reported rather than refused")
	flag.Parse()

	origin := commit.Internal
	if *external {
		origin = commit.External
	}

	result, err := commit.Run(os.Stdout, ".", *doc, *base, *head, origin)
	switch {
	case err != nil:
		fmt.Fprintf(os.Stderr, "commithygiene: %v\n", err)
		os.Exit(statusBroken)
	case result.NoRange != "":
		fmt.Fprintf(os.Stderr, "commithygiene: %s, so nothing was judged.\n", result.NoRange)
		os.Exit(statusNoRange)
	case commit.Failed(result.Findings):
		fmt.Fprintf(os.Stdout, "\nThe rules are in %s, and widening one is an edit there.\n", *doc)
		os.Exit(statusFail)
	}
	os.Exit(statusPass)
}
