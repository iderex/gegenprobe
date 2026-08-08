package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goodFixture is the file every case below changes one line of. It is written
// as lines so that a case can substitute exactly one of them and the near miss
// is a near miss rather than a different file.
func goodFixtureLines() []string {
	return []string{
		"Code: GRASP2018",
		"Version: 2018.1",
		"Case: hydrogen-like-1s2p",
		"Kept: the first two lines of the level table",
		"",
		"aGVsbG8K",
		"",
	}
}

func goodFixture() string { return strings.Join(goodFixtureLines(), "\n") }

// fixtureTree writes a tree holding the named files, so the leg can walk
// something shaped like this repository.
func fixtureTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestAFixtureInTheConventionPasses(t *testing.T) {
	root := fixtureTree(t, map[string]string{"internal/reader/testdata/levels.fixture": goodFixture()})

	if o := fixtureStorageLeg(root); o.verdict != passed {
		t.Fatalf("a fixture in the convention was refused: %s", o.detail)
	}
}

// The one line mistake somebody actually makes, which is storing the bytes as
// the file they arrived as.
func TestAFixtureStoredRawIsRefusedAndTheRepairIsNamed(t *testing.T) {
	root := fixtureTree(t, map[string]string{"internal/reader/testdata/levels.out": "  1  -1  1s\r\n"})

	o := fixtureStorageLeg(root)

	if o.verdict != failed {
		t.Fatalf("a raw file under testdata passed: %v %s", o.verdict, o.detail)
	}
	for _, want := range []string{"internal/reader/testdata/levels.out", "stored raw", conventionDoc} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the failure does not carry %q:\n%s", want, o.detail)
		}
	}
}

func TestEachWayTheProvenanceIsWrongIsRefused(t *testing.T) {
	for _, c := range []struct {
		name   string
		mutate func([]string) []string
		says   string
	}{
		{"a missing field", func(l []string) []string { return append(l[:3:3], l[4:]...) }, "missing Kept"},
		{"an empty field", func(l []string) []string { l[3] = "Kept:"; return l }, "Kept is empty"},
		{"a misspelt field", func(l []string) []string { l[0] = "Codee: GRASP2018"; return l }, "not a fixture field"},
		{"a payload that is not base64", func(l []string) []string { l[5] = "  1  -1  1s"; return l }, "not base64"},
		{"no blank line", func(l []string) []string { return append(l[:4:4], l[5:]...) }, "no blank line"},
	} {
		t.Run(c.name, func(t *testing.T) {
			lines := c.mutate(goodFixtureLines())
			root := fixtureTree(t, map[string]string{"internal/reader/testdata/levels.fixture": strings.Join(lines, "\n")})

			o := fixtureStorageLeg(root)

			if o.verdict != failed {
				t.Fatalf("%s passed: %s", c.name, o.detail)
			}
			if !strings.Contains(o.detail, c.says) {
				t.Errorf("the failure does not say why:\n%s", o.detail)
			}
			if !strings.Contains(o.detail, "levels.fixture") {
				t.Errorf("the failure does not name the file:\n%s", o.detail)
			}
		})
	}
}

// legsThatJudgeAFixtureTree is the legs that can say anything about a tree of
// this shape. The language legs shell out to the toolchain and would fail on any
// such tree for a reason that has nothing to do with these fixtures, so they are
// not in the list. That is the bound on the case below: it shows the fixture
// storage leg is what refuses each fixture among the legs that examine it, not
// among every leg the command has.
func legsThatJudgeAFixtureTree(withFixtureStorage bool) []leg {
	ls := []leg{
		{name: "format", subject: "fixture", run: formatLeg},
		{name: "action pinning", subject: "fixture", run: actionPinningLeg},
	}
	if withFixtureStorage {
		ls = append(ls, leg{name: "fixture storage", subject: "fixture", run: fixtureStorageLeg})
	}
	return ls
}

// Disabling the leg has to turn every fixture green. A fixture still refused
// with the leg removed is a fixture proving something else.
func TestDisablingTheFixtureStorageLegTurnsItsFixturesGreen(t *testing.T) {
	for _, c := range []struct{ name, path, body string }{
		{"stored raw", "internal/reader/testdata/levels.out", "  1  -1  1s\r\n"},
		{"a missing field", "internal/reader/testdata/levels.fixture",
			strings.Join(append(goodFixtureLines()[:3:3], goodFixtureLines()[4:]...), "\n")},
		{"outside testdata", "internal/reader/levels.fixture", goodFixture()},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := fixtureTree(t, map[string]string{c.path: c.body})

			if status := run(io.Discard, root, legsThatJudgeAFixtureTree(true)); status == 0 {
				t.Fatalf("the gate passed %s with the leg enabled", c.name)
			}
			if status := run(io.Discard, root, legsThatJudgeAFixtureTree(false)); status != 0 {
				t.Errorf("the gate still refused %s with the leg disabled, so the fixture proves something else", c.name)
			}
		})
	}
}

func TestAFixtureOutsideATestdataDirectoryIsRefused(t *testing.T) {
	root := fixtureTree(t, map[string]string{"internal/reader/levels.fixture": goodFixture()})

	o := fixtureStorageLeg(root)

	if o.verdict != failed {
		t.Fatalf("a fixture nothing would look for passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "not where anything looks for one") {
		t.Errorf("the failure does not say why:\n%s", o.detail)
	}
}

// Where there is nothing to read the leg says so rather than passing, because a
// green line standing for no examination is the shape this command exists to
// refuse.
func TestTheLegSkipsWhereThereIsNoFixtureToRead(t *testing.T) {
	o := fixtureStorageLeg(fixtureTree(t, map[string]string{"internal/reader/reader.go": "package reader\n"}))

	if o.verdict != skipped {
		t.Fatalf("a tree with no testdata directory did not report a skip: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "no fixture was read") {
		t.Errorf("the skip does not say what was not examined: %q", o.detail)
	}
}

// A carriage return inside a fixture is the byte the convention exists to keep,
// and the leg has to accept it rather than treat it as damage. The bytes reach
// the file as base64, so nothing about them is visible to a line ending rule.
func TestAFixtureWhoseBytesHoldCarriageReturnsIsAccepted(t *testing.T) {
	body := "Code: hand-written\nVersion: not applicable\nCase: not applicable\nKept: everything\n\nDQoNCg==\n"
	root := fixtureTree(t, map[string]string{"internal/reader/testdata/crlf.fixture": body})

	if o := fixtureStorageLeg(root); o.verdict != passed {
		t.Fatalf("a fixture carrying carriage returns was refused: %s", o.detail)
	}
}
