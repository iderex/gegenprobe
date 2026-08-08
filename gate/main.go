// Command gate runs every check this repository has, in a fixed order, and stops
// at the first failure. There is one way to find out whether the tree is in a
// good state and it is this command, so that what a contributor runs before
// pushing and what runs afterwards cannot drift apart.
//
// What it prints is the whole account of the run. Every leg names itself and its
// verdict, a leg that could not run says what was missing, and a run that stopped
// early names the legs that never ran. A run covering less than the whole set
// therefore cannot be read as one that covered it and found nothing.
//
// A later check is added by adding a leg in legs.go and nowhere else. Anything
// that can only be run by a workflow is a check that cannot be run before
// pushing, and this repository is not going to have one.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// root is the directory the legs judge. The command is invoked as
// "go run ./gate", which resolves only from the repository root, so the working
// directory is the root by construction rather than by search.
const root = "."

func main() {
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		fmt.Fprintln(os.Stderr, "gate: no go.mod in the working directory; run this from the repository root: go run ./gate")
		os.Exit(1)
	}
	os.Exit(run(os.Stdout, root, legs()))
}
