package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// The tests in this file are the ones that need a built binary or the files
// beside it, so they run from the repository root, which is where the go tool
// puts a test binary's working directory for the root package.

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

	mentioned := map[string]bool{}
	for _, m := range versionInProse.FindAllStringSubmatch(string(record), -1) {
		mentioned[m[1]] = true
	}
	if len(mentioned) == 0 {
		t.Fatalf("%s mentions no Go version at all, so the record no longer says what the floor is", matches[0])
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
			declared, matches[0], strings.Join(wrong, ", "))
	}
}

// build compiles the command into dir and returns the path to the binary. It
// takes the same route an operator takes, so what is asserted below is the
// program rather than a function that happens to be called by it.
func build(t *testing.T, dir string) string {
	t.Helper()
	out := filepath.Join(dir, "gegenprobe")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", out, ".")
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build -o %s .: %v\n%s", out, err, combined)
	}
	return out
}

func TestBuiltBinaryPrintsAVersion(t *testing.T) {
	out, err := exec.Command(build(t, t.TempDir()), "version").Output()
	if err != nil {
		t.Fatalf("running the binary with `version`: %v", err)
	}
	if strings.TrimSpace(string(out)) == "" {
		t.Error("`version` printed nothing, and an empty version is the one answer a bug report cannot use")
	}
}

func TestBuiltBinaryWithNoArgumentsPrintsUsageAndExitsNonZero(t *testing.T) {
	cmd := exec.Command(build(t, t.TempDir()))
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("the binary exited zero with no arguments, want non zero")
	}
	if !strings.Contains(string(combined), "usage:") {
		t.Errorf("output with no arguments = %q, want the usage text", combined)
	}
}

// Two builds of one commit have to agree on the version, which is what makes
// the string usable as an identifier at all. This is the version half of the
// property; the byte for byte half over the binary itself is #28.
func TestTwoBuildsOfTheSameCommitAgreeOnTheVersion(t *testing.T) {
	first, err := exec.Command(build(t, t.TempDir()), "version").Output()
	if err != nil {
		t.Fatal(err)
	}
	second, err := exec.Command(build(t, t.TempDir()), "version").Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("two builds of this commit reported %q and %q; a version that moves between builds identifies nothing",
			strings.TrimSpace(string(first)), strings.TrimSpace(string(second)))
	}
}
