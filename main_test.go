package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// languageDecisionRecord is the record the language decision issue fixes by
// name. The minimum Go version lives there and in go.mod, and this test is what
// stops the two from drifting apart.
const languageDecisionRecord = "docs/decisions/0001-the-harness-is-written-in-go.md"

// goDirective matches the go line of a module file and nothing else. A godebug
// or a toolchain line does not match, because the token has to end at the space.
var goDirective = regexp.MustCompile(`(?m)^go[ \t]+(\S+)[ \t\r]*$`)

// goVersionInText matches every Go version the record names, in prose and in
// the module file it quotes, so a record naming two different numbers is caught
// as well as a record disagreeing with go.mod.
var goVersionInText = regexp.MustCompile(`(?i)\bgo[ \t]+(\d+\.\d+(?:\.\d+)?)\b`)

func TestGoDirectiveMatchesTheLanguageDecisionRecord(t *testing.T) {
	mod, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("reading go.mod: %v", err)
	}
	m := goDirective.FindSubmatch(mod)
	if m == nil {
		t.Fatal("go.mod carries no go directive")
	}
	declared := string(m[1])

	record, err := os.ReadFile(filepath.FromSlash(languageDecisionRecord))
	if err != nil {
		t.Fatalf("the minimum Go version is fixed by %s and go.mod restates it as %s; the record could not be read: %v",
			languageDecisionRecord, declared, err)
	}

	named := goVersionInText.FindAllSubmatch(record, -1)
	if len(named) == 0 {
		t.Fatalf("%s names no Go version, and go.mod says go %s", languageDecisionRecord, declared)
	}
	for _, n := range named {
		if got := string(n[1]); got != declared {
			t.Errorf("%s names Go %s; go.mod says go %s", languageDecisionRecord, got, declared)
		}
	}
}

// builtCommand is linked once for the whole package. Linking is the expensive
// part of this suite, so a test that does not specifically need a second binary
// uses this one.
var builtCommand string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gegenprobe-build")
	if err != nil {
		panic(err)
	}
	builtCommand, err = build(dir)
	if err != nil {
		os.RemoveAll(dir)
		panic(err)
	}
	status := m.Run()
	os.RemoveAll(dir)
	os.Exit(status)
}

// The built binary is what an operator has, so the two conditions about what it
// prints are checked on the binary rather than on the package behind it.
func TestBuiltCommandWithoutArgumentsPrintsUsageAndFails(t *testing.T) {
	stdout, stderr, status := runBuilt(t, builtCommand)
	if status == 0 {
		t.Error("the command succeeded with no arguments; it has to fail")
	}
	if !strings.Contains(stderr, "usage: gegenprobe") {
		t.Errorf("no usage on stderr, got %q", stderr)
	}
	if stdout != "" {
		t.Errorf("a misuse wrote to stdout: %q", stdout)
	}
}

func TestBuiltCommandPrintsAVersion(t *testing.T) {
	stdout, stderr, status := runBuilt(t, builtCommand, "version")
	if status != 0 {
		t.Errorf("version exited %d, stderr %q", status, stderr)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Error("version printed nothing")
	}
}

// Nothing in the version string may come from the clock, the build host or a
// path on it. A second, independent build of the same tree is the cheapest way
// to prove that, and it is the property the reproducible build work rests on.
func TestASecondBuildOfTheSameTreeAgreesOnTheVersion(t *testing.T) {
	second, err := build(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	first, _, status := runBuilt(t, builtCommand, "version")
	if status != 0 {
		t.Fatalf("the first build exited %d on version", status)
	}
	again, _, status := runBuilt(t, second, "version")
	if status != 0 {
		t.Fatalf("the second build exited %d on version", status)
	}
	if first != again {
		t.Errorf("two builds of one tree printed %q then %q", first, again)
	}
}

func build(dir string) (string, error) {
	name := "gegenprobe"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	out, err := exec.Command("go", "build", "-o", path, ".").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go build: %w\n%s", err, out)
	}
	return path, nil
}

func runBuilt(t *testing.T, bin string, args ...string) (stdout, stderr string, status int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err := cmd.Run()
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("running %s: %v", bin, err)
		}
		status = exit.ExitCode()
	}
	return out.String(), errOut.String(), status
}
