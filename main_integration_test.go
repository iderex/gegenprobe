//go:build integration

package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The tests here build the command and run it, which is an import of os/exec,
// and record 0009 puts that at the top of the list the gate tier may not have.
// One rule covers elevation, installers, service registration, certificate
// trust stores, container engines and every external binary, rather than a list
// of program names that would always be one name short, so it covers a compiler
// too. They therefore run under the `integration` tag and not before it is
// asked for.
//
// What the gate tier no longer asserts, and nobody should read the green line
// below as covering. Nothing in the gate tier now says that the built binary
// prints a version, that it prints usage and exits non zero with no arguments,
// or that two builds of one commit report the same version. Those are
// statements about a program, and the tier that runs everywhere cannot make
// them without starting one.
//
// What is asserted in the gate tier instead is the same behaviour one call
// short of a process: TestVersionPrintsAVersionOnStdoutAndSucceeds and
// TestNoArgumentsPrintsUsageAndFails in internal/cli cover the text and the
// status over cli.Run, and TestResolveIsStable in internal/version covers a
// version that does not move between calls. That is not the same property. A
// linker flag, a build tag, a wrapper added to main, or a version stamped from
// something that varies would break the binary and leave all three green.
//
// The other route this could have taken was a successor to record 0009 carving
// out the toolchain. It was not taken: 0009 says a change that makes a gate
// test need a forbidden capability is a defect in the change rather than a
// reason to widen the list, and its own tie break puts a contested test in the
// gate with whatever it wanted turned into a fixture. A built binary cannot be
// turned into a fixture, so the impossibility is the answer and this comment is
// where the record asks for it to be written.

// requireToolchain refuses to let this file pass quietly on a machine that
// cannot run it. Record 0009 asks this tier to print, per precondition, that it
// was not asked for and what asking would have cost, because a suite that
// returns success when its preconditions are missing is green where it matters
// least.
func requireToolchain(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("no `go` on PATH, so nothing here built or ran the command and this file asserted nothing. "+
			"Asking for it costs a Go toolchain at the version go.mod declares and a writable build directory: %v", err)
	}
}

// build compiles the command into dir and returns the path to the binary. It
// takes the same route an operator takes, so what is asserted below is the
// program rather than a function that happens to be called by it.
func build(t *testing.T, dir string) string {
	t.Helper()
	requireToolchain(t)
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
