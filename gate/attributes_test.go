package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iderex/gegenprobe/internal/fixture"
)

// The passing pair every case below changes one line of: a declaration naming
// three types, and the tracked paths of a tree holding exactly those three.
func goodDeclaration() string {
	return strings.Join([]string{
		"# what git may do here",
		"* -text",
		"",
		"*.go        text eol=lf",
		"*.md        text eol=lf",
		"*.fixture   -text",
	}, "\n")
}

func goodTracked() []string {
	return []string{"README.md", "gate/legs.go", "internal/fixture/testdata/carriage-return.fixture"}
}

func TestThePassingPairAgreesAboutEveryType(t *testing.T) {
	o := judgeTypes([]byte(goodDeclaration()), goodTracked())

	if o.verdict != passed {
		t.Fatalf("a declaration naming every tracked type was refused: %s", o.detail)
	}
	if !strings.Contains(o.detail, "3 type(s)") {
		t.Errorf("the pass does not say what it examined: %q", o.detail)
	}
}

// The one line mistake somebody actually makes: a new kind of file arrives and
// nothing is written for it. The declaration below differs from the passing one
// by exactly the line that would have decided it.
func withoutTheFixtureLine() string {
	var kept []string
	for _, line := range strings.Split(goodDeclaration(), "\n") {
		if strings.HasPrefix(line, "*.fixture") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func TestATypeNoPatternNamesIsRefused(t *testing.T) {
	o := judgeTypes([]byte(withoutTheFixtureLine()), goodTracked())

	if o.verdict != failed {
		t.Fatalf("a tracked type nothing declares passed: %v %s", o.verdict, o.detail)
	}
	for _, want := range []string{".fixture", "internal/fixture/testdata/carriage-return.fixture", "1 tracked file(s)"} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the failure does not carry %q:\n%s", want, o.detail)
		}
	}
}

func TestTheTwoDeclarationsDifferByExactlyOneLine(t *testing.T) {
	good := strings.Split(goodDeclaration(), "\n")
	bad := strings.Split(withoutTheFixtureLine(), "\n")

	if len(good)-len(bad) != 1 {
		t.Fatalf("the declarations differ by %d lines; a near miss one line apart proves more", len(good)-len(bad))
	}
}

// A catch-all is a default and not a decision. Counting it would let a tree
// declare nothing about anything and pass, which is the state this leg exists
// to replace.
func TestACatchAllNamesNoType(t *testing.T) {
	o := judgeTypes([]byte("* -text\n"), []string{"main.go"})

	if o.verdict != failed {
		t.Fatalf("a tree whose only rule is a catch-all passed: %s", o.detail)
	}
	if !strings.Contains(o.detail, ".go") {
		t.Errorf("the failure does not name the type that was not decided:\n%s", o.detail)
	}
}

func TestWhatNamesATypeAndWhatDoesNot(t *testing.T) {
	for _, c := range []struct{ pattern, want string }{
		{"*.go", ".go"},
		{".gitignore", ".gitignore"},
		{"LICENSE", "LICENSE"},
		{"*", ""},
		{"**", ""},
		{"*.", ""},
		{"*.f*", ""},
		{"**/testdata/**", ""},
		{"docs/*.md", ""},
	} {
		t.Run(c.pattern, func(t *testing.T) {
			if got := typeNamedBy(c.pattern); got != c.want {
				t.Errorf("%q names %q, and %q was expected", c.pattern, got, c.want)
			}
		})
	}
}

// A rule under a directory refines what a tree-wide rule decided and cannot
// stand in for one. Reading it as a decision about the whole tree is how a check
// comes to disagree with the thing it checks.
func TestADirectoryRuleDoesNotCoverTheTypeElsewhere(t *testing.T) {
	o := judgeTypes([]byte("**/testdata/** -text\n"), []string{"gate/testdata/a.fixture"})

	if o.verdict != failed {
		t.Fatalf("a directory rule was read as a decision about a type: %s", o.detail)
	}
}

func TestACommentAMacroAndABlankLineNameNothing(t *testing.T) {
	named := namedTypes([]byte("# *.go text\n\n[attr]binary -text\n  \n*.md text\n"))

	if named[".go"] {
		t.Error("a pattern inside a comment was read as a declaration")
	}
	if len(named) != 1 || !named[".md"] {
		t.Errorf("the declaration names %v, and only .md was written outside a comment", named)
	}
}

func TestFileTypeIsTheExtensionOrTheWholeName(t *testing.T) {
	for _, c := range []struct{ path, want string }{
		{"gate/legs.go", ".go"},
		{"go.mod", ".mod"},
		{".gitignore", ".gitignore"},
		{".github/workflows/ci.yml", ".yml"},
		{"NOTICE.md", ".md"},
		{"LICENSE", "LICENSE"},
		{"a/b/archive.tar.gz", ".gz"},
	} {
		t.Run(c.path, func(t *testing.T) {
			if got := fileType(c.path); got != c.want {
				t.Errorf("%q is of type %q, and %q was expected", c.path, got, c.want)
			}
		})
	}
}

// Where there is no declaration the leg refuses rather than skipping. A skip
// here would be a green line standing for the exact state the leg exists to
// refuse: a tree in which every file is converted by somebody's local setting.
func TestTheAttributesLegRefusesATreeWithNoDeclaration(t *testing.T) {
	o := attributesLeg(t.TempDir())

	if o.verdict != failed {
		t.Fatalf("a tree with no declaration did not fail: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "core.autocrlf") {
		t.Errorf("the failure does not say what the absence costs:\n%s", o.detail)
	}
}

// Where git cannot say what is tracked, the leg reports that it read nothing.
// A directory that is not a repository is the shape a released archive has.
func TestTheAttributesLegSaysSoWhereNothingCanBeListed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, attributesFile), []byte(goodDeclaration()), 0o644); err != nil {
		t.Fatal(err)
	}

	o := attributesLeg(dir)

	if o.verdict != skipped {
		t.Fatalf("a directory git cannot read was judged rather than skipped: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "no file type was read") {
		t.Errorf("the skip does not say what was not examined: %q", o.detail)
	}
}

func TestTheRecordedBytesLegSaysSoWhereThereIsNoDeclaration(t *testing.T) {
	o := recordedBytesLeg(t.TempDir())

	if o.verdict != skipped {
		t.Fatalf("a tree with no declaration was judged rather than skipped: %v %s", o.verdict, o.detail)
	}
}

// The probe is the whole of what the round trip measures, so a probe holding no
// carriage return would turn that leg into one that passes by construction.
func TestTheProbeHoldsCarriageReturnsAndIsItselfAFixture(t *testing.T) {
	if strings.Count(probeFixture, "\r") == 0 {
		t.Fatal("the probe holds no carriage return, so taking it through git proves nothing")
	}
	if _, problems := fixture.Parse([]byte(probeFixture)); len(problems) > 0 {
		t.Errorf("the probe is not stored in the form docs/fixtures.md fixes: %s", strings.Join(problems, "; "))
	}
}

func TestMovedBytesIsSilentWhereNothingMovedAndCountsWhereItDid(t *testing.T) {
	if got := movedBytes("git add", []byte("a\r\nb\r\n"), []byte("a\r\nb\r\n")); got != "" {
		t.Errorf("an unchanged round trip was reported as a change: %q", got)
	}

	got := movedBytes("git add", []byte("a\r\nb\r\n"), []byte("a\nb\n"))
	for _, want := range []string{"git add", "6 bytes in with 2 carriage return(s)", "4 bytes out with 0"} {
		if !strings.Contains(got, want) {
			t.Errorf("the report does not carry %q: %q", want, got)
		}
	}
}
