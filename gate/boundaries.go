package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/iderex/gegenprobe/internal/boundary"
)

// The direction the packages here depend in is a decision, and until now it was
// a decision nothing held. The lint that reads source text catches an import
// somebody typed in the wrong file; this reads the graph the toolchain reports,
// so an edge arriving through a rename or a move is seen the same way.
//
// The declaration is a document rather than a table in this file, for the reason
// every other declaration here is: the boundary is an argument somebody has to
// make, and it belongs next to the decision it comes from.
const boundaryDoc = "docs/package-boundaries.md"

func packageBoundariesLeg(root string) outcome {
	declaration, err := os.ReadFile(filepath.Join(root, boundaryDoc))
	if err != nil {
		return fail(err.Error())
	}
	entries, err := boundary.ParseDeclaration(string(declaration))
	if err != nil {
		return fail(boundaryDoc + ": " + err.Error())
	}

	packages, err := boundary.Graph(root)
	if err != nil {
		return fail(err.Error())
	}

	violations := boundary.Conform(packages, entries)
	if len(violations) > 0 {
		lines := make([]string, 0, len(violations)+2)
		for _, v := range violations {
			lines = append(lines, v.String())
		}
		lines = append(lines, "", "The permitted direction is "+boundaryDoc+
			", and widening it is an edit there beside the decision it comes from.")
		return fail(strings.Join(lines, "\n"))
	}

	edges := 0
	for _, p := range packages {
		edges += len(p.Imports) + len(p.TestImports)
	}
	return note(fmt.Sprintf("%d package(s) read and %d edge(s) inside this module, every one of them declared in %s.",
		len(packages), edges, boundaryDoc))
}
