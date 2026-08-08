package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// This file carries no build constraint, so it is a gate tier file and record
// 0009's list of forbidden capabilities applies to it. What is left here reads
// files beside the command from a relative path, which the tier permits, and
// runs from the repository root, which is where the go tool puts a test
// binary's working directory for the root package.
//
// The tests that built the command and ran it moved to
// main_integration_test.go, under the `integration` tag, because starting a
// program is an import of os/exec and 0009 puts that at the top of what this
// tier may not have. What the gate no longer asserts is written down there.

// goDirective is the `go 1.24` line in go.mod. The module file is the one place
// the floor is declared, per record 0001.
var goDirective = regexp.MustCompile(`(?m)^go (\d+(?:\.\d+)+)\s*$`)

// versionInProse is any mention of a Go version anywhere in a record, in either
// case, so that a second mention added later is checked rather than ignored.
var versionInProse = regexp.MustCompile(`(?i)\bgo (\d+(?:\.\d+)+)`)

// Record 0001 fixes the minimum Go version and says it is declared in go.mod
// rather than restated. The record still has to name a number for a reader, so
// the two can disagree, and a person noticing that is not a mechanism. This is.
//
// It checks every version the record mentions rather than the first, because
// the way this drifts is a raise that updates one of the two mentions.
func TestGoDirectiveMatchesTheDecisionRecord(t *testing.T) {
	gomod, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	m := goDirective.FindSubmatch(gomod)
	if m == nil {
		t.Fatal("go.mod carries no `go` directive, so there is no floor to compare against")
	}
	declared := string(m[1])

	matches, err := filepath.Glob(filepath.Join("docs", "decisions", "0001-*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("found %d files matching docs/decisions/0001-*.md, want exactly 1: %v", len(matches), matches)
	}
	record, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	// The message names a path a reader will retype, so it is written with the
	// separator the tree uses rather than the one this platform happens to.
	recordPath := filepath.ToSlash(matches[0])

	mentioned := map[string]bool{}
	for _, m := range versionInProse.FindAllStringSubmatch(string(record), -1) {
		mentioned[m[1]] = true
	}
	if len(mentioned) == 0 {
		t.Fatalf("%s mentions no Go version at all, so the record no longer says what the floor is", recordPath)
	}

	var wrong []string
	for v := range mentioned {
		if v != declared {
			wrong = append(wrong, v)
		}
	}
	sort.Strings(wrong)
	if len(wrong) > 0 {
		t.Errorf("go.mod declares Go %s and %s mentions %s. Raising the floor is a change to go.mod and to a successor record, so make the two agree.",
			declared, recordPath, strings.Join(wrong, ", "))
	}
}
