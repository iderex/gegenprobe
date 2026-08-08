package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The builder is a parameter, so every case below is a pair of artefacts this
// test chose rather than a compiler run. That is what keeps these in the gate
// tier: record 0009 refuses os/exec here, and a leg whose proof needs a
// toolchain is a leg that is proven on some machines and not others.

// writes returns a builder that produces the given bytes, one build at a time,
// so a case can say what the first build produced and what the second did.
func writes(bodies ...string) buildFunc {
	call := 0
	return func(_, out string) error {
		body := bodies[len(bodies)-1]
		if call < len(bodies) {
			body = bodies[call]
		}
		call++
		return os.WriteFile(out, []byte(body), 0o644)
	}
}

// judge runs the leg over a tree this test owns, with a builder this test owns.
func judge(t *testing.T, build buildFunc) outcome {
	t.Helper()
	return reproducibleBuild(t.TempDir(), t.TempDir(), build)
}

func TestTwoBuildsThatAgreeAreAPassAndTheNoteCarriesTheChecksum(t *testing.T) {
	o := judge(t, writes("the same bytes twice"))

	if o.verdict != passed {
		t.Fatalf("two identical builds were refused: %s", o.detail)
	}
	if !strings.Contains(o.detail, sum([]byte("the same bytes twice"))) {
		t.Errorf("the note does not carry the checksum a reader would compare against:\n%s", o.detail)
	}
	if !strings.Contains(o.detail, "same tree twice") {
		t.Errorf("the note does not say which of the two properties it proved:\n%s", o.detail)
	}
}

// The one-character mistake this leg is for: something in the build reads a
// value that moves, and the artefact stops being checkable against a published
// checksum.
func TestABuildThatMovesBetweenRunsIsRefusedAndNamesBothChecksums(t *testing.T) {
	o := judge(t, writes("built at 12:00", "built at 12:01"))

	if o.verdict != failed {
		t.Fatalf("a build whose bytes moved between runs passed: %v %s", o.verdict, o.detail)
	}
	for _, want := range []string{sum([]byte("built at 12:00")), sum([]byte("built at 12:01"))} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the failure does not name %s, so a reader cannot tell which build to look at:\n%s", want, o.detail)
		}
	}
}

// -trimpath produces no output when it is forgotten, so this is the only thing
// that says it was passed. Both spellings, because the tree is developed on one
// separator and built on the other.
func TestABinaryCarryingTheBuildDirectoryIsRefusedAndTheFlagIsNamed(t *testing.T) {
	for _, c := range []struct {
		name  string
		write func(root string) string
	}{
		{"forward slashes", func(root string) string { return "\x00padding" + filepath.ToSlash(root) + "/internal/cli\x00" }},
		{"backslashes", func(root string) string {
			return "\x00padding" + strings.ReplaceAll(filepath.ToSlash(root), "/", `\`) + `\internal\cli` + "\x00"
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()

			o := reproducibleBuild(root, t.TempDir(), writes(c.write(root)))

			if o.verdict != failed {
				t.Fatalf("a binary holding the build directory passed: %v %s", o.verdict, o.detail)
			}
			if !strings.Contains(o.detail, "-trimpath") {
				t.Errorf("the failure does not name the flag that repairs it:\n%s", o.detail)
			}
		})
	}
}

// A binary holding an absolute path that is not the build directory is not this
// failure, and a leg refusing it would be one somebody switches off. The near
// miss is one component away: same parent, different leaf.
func TestAPathOneComponentAwayFromTheBuildDirectoryIsNotRefused(t *testing.T) {
	root := t.TempDir()
	neighbour := filepath.ToSlash(filepath.Dir(root)) + "/a-different-directory/x"

	o := reproducibleBuild(root, t.TempDir(), writes("\x00"+neighbour+"\x00"))

	if o.verdict != passed {
		t.Fatalf("a path that is not the build directory was refused: %s", o.detail)
	}
}

func TestABuilderThatFailsIsAFailureRatherThanAPass(t *testing.T) {
	o := judge(t, func(_, _ string) error { return os.ErrPermission })

	if o.verdict != failed {
		t.Fatalf("a build that never ran reported %v, want a failure", o.verdict)
	}
	if !strings.Contains(o.detail, "first build") {
		t.Errorf("the failure does not say which of the two builds it was:\n%s", o.detail)
	}
}

// A builder that reports success and writes nothing is the shape that would
// otherwise compare two absent files and find them equal.
func TestABuilderThatWritesNothingIsRefused(t *testing.T) {
	o := judge(t, func(_, _ string) error { return nil })

	if o.verdict != failed {
		t.Fatalf("a build that wrote no artefact reported %v, want a failure", o.verdict)
	}
}

func TestAnEmptyArtefactIsRefused(t *testing.T) {
	o := judge(t, writes(""))

	if o.verdict != failed {
		t.Fatalf("two empty files compared equal and passed: %v %s", o.verdict, o.detail)
	}
}

// Where there is no toolchain the leg says what was missing rather than passing,
// because a green line standing for no examination is what this command exists
// to refuse.
func TestTheLegSkipsWhereThereIsNoToolchain(t *testing.T) {
	t.Setenv("PATH", "")

	o := reproducibleBuildLeg(t.TempDir())

	if o.verdict != skipped {
		t.Fatalf("with no toolchain reachable the leg reported %v, want a skip: %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "built the command twice") {
		t.Errorf("the skip does not say what was not examined: %q", o.detail)
	}
}

// legsThatJudgeABuild is the run with and without this leg, so that a fixture
// still refused once the leg is gone can be recognised as one proving something
// else.
func legsThatJudgeABuild(t *testing.T, withReproducibleBuild bool, build buildFunc) []leg {
	t.Helper()
	workdir := t.TempDir()
	ls := []leg{
		{name: "format", subject: "fixture", run: formatLeg},
	}
	if withReproducibleBuild {
		ls = append(ls, leg{
			name:    "reproducible build",
			subject: "fixture",
			run:     func(root string) outcome { return reproducibleBuild(root, workdir, build) },
		})
	}
	return ls
}

func TestDisablingTheReproducibleBuildLegTurnsItsFixturesGreen(t *testing.T) {
	for _, c := range []struct {
		name  string
		build func(root string) buildFunc
	}{
		{"bytes that moved", func(string) buildFunc { return writes("built at 12:00", "built at 12:01") }},
		{"the build directory in the artefact", func(root string) buildFunc {
			return writes("\x00" + filepath.ToSlash(root) + "/internal/cli\x00")
		}},
		{"nothing written", func(string) buildFunc { return func(_, _ string) error { return nil } }},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()

			if status := run(io.Discard, root, legsThatJudgeABuild(t, true, c.build(root))); status == 0 {
				t.Fatalf("the gate passed %s with the leg enabled", c.name)
			}
			if status := run(io.Discard, root, legsThatJudgeABuild(t, false, c.build(root))); status != 0 {
				t.Errorf("the gate still refused %s with the leg disabled, so the fixture proves something else", c.name)
			}
		})
	}
}
