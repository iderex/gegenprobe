package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The judging this leg wraps is tested over recorded graphs in internal/boundary.
// What is left here is the leg's own three ways of not reaching a verdict, and
// each is a failure rather than a pass, because a leg that could not read the
// declaration has judged nothing and must not say the tree conforms.

func TestTheLegRefusesATreeWithNoDeclaration(t *testing.T) {
	o := packageBoundariesLeg(t.TempDir())
	if o.verdict != failed {
		t.Fatalf("a tree with no declaration was not refused: %v", o)
	}
	if !strings.Contains(o.detail, "package-boundaries") {
		t.Errorf("the failure does not name what it could not read: %q", o.detail)
	}
}

func TestTheLegRefusesADeclarationThatPlacesNothing(t *testing.T) {
	root := treeWithDeclaration(t, "# a document that places nothing\n")
	o := packageBoundariesLeg(root)
	if o.verdict != failed {
		t.Fatalf("a declaration holding no entry was not refused: %v", o)
	}
	if !strings.Contains(o.detail, "no entry") {
		t.Errorf("the failure does not say what is missing: %q", o.detail)
	}
}

// A readable declaration over a directory the toolchain cannot list is the third
// way out, and it is the one that would otherwise be reported as a tree with no
// edges to refuse.
func TestTheLegRefusesATreeTheToolchainCannotList(t *testing.T) {
	root := treeWithDeclaration(t, "## gate\n\nDecision: 0009\nMay-import: nothing\nMay-import-in-tests: nothing\n")
	o := packageBoundariesLeg(root)
	if o.verdict != failed {
		t.Fatalf("a directory that is not a module was not refused: %v", o)
	}
}

func TestThisTreeConformsToItsOwnDeclaration(t *testing.T) {
	o := packageBoundariesLeg("..")
	if o.verdict != passed {
		t.Fatalf("this tree does not conform to its own declaration: %v", o)
	}
	if !strings.Contains(o.detail, "edge(s) inside this module") {
		t.Errorf("the pass does not say what it covered: %q", o.detail)
	}
}

func treeWithDeclaration(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("making the tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, boundaryDoc), []byte(body), 0o644); err != nil {
		t.Fatalf("writing the declaration: %v", err)
	}
	return root
}
