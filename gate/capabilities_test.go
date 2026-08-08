package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// absolutePaths reads the two path literals the cases below need. They are held
// under testdata rather than written into this file, because a literal here
// would be a gate tier test holding an absolute path, which is the thing the
// leg refuses and is right to refuse. Record 0009 asks for a fixture rather than
// a suppression where a rule catches something legitimate, and this is it.
func absolutePaths(t *testing.T) []string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "absolute-path-literals.txt"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	if len(out) < 2 {
		t.Fatalf("the fixture carries %d path(s); the cases below need a POSIX one and a Windows one", len(out))
	}
	return out
}

// The fixtures are one template with two slots, an import line and a statement
// line, and every case below substitutes exactly one of them. A verdict that
// moves between the passing fixture and a refused one therefore moved for the
// line the case is about and not for anything around it.
//
// The template's line numbering is fixed and the cases assert on it: the import
// slot is line 5 and the statement slot is line 10.
func tierSource(importLine, statementLine string) string {
	return "package x\n" +
		"\n" +
		"import (\n" +
		"\t\"testing\"\n" +
		"\t" + importLine + "\n" +
		")\n" +
		"\n" +
		"func TestSomething(t *testing.T) {\n" +
		"\tdir := t.TempDir()\n" +
		"\t" + statementLine + "\n" +
		"\t_ = dir\n" +
		"}\n"
}

const (
	passingImport    = "\"strings\""
	passingStatement = "_ = strings.TrimSpace(\" \")"

	importSlotLine    = 5
	statementSlotLine = 10
)

func passingTierSource() string { return tierSource(passingImport, passingStatement) }

// capabilityFixtures pairs every item on the refusable list with the one line that trips
// it. A test below asserts this covers items() exactly, so an item added to the
// list without a fixture reddens rather than passing unproven.
func capabilityFixtures(t *testing.T) map[string]struct {
	src  string
	line int
	says string
} {
	t.Helper()
	posix := absolutePaths(t)[0]
	return map[string]struct {
		src  string
		line int
		says string
	}{
		itemExec:        {tierSource("\"os/exec\"", passingStatement), importSlotLine, "imports os/exec"},
		itemNetwork:     {tierSource("\"net/http\"", passingStatement), importSlotLine, "reaches the network"},
		itemOutside:     {tierSource("\"golang.org/x/term\"", passingStatement), importSlotLine, "outside the standard library"},
		itemSyscall:     {tierSource("\"syscall\"", passingStatement), importSlotLine, "privilege request"},
		itemDisplay:     {tierSource(passingImport, "_ = os.Getenv(\"DISPLAY\")"), statementSlotLine, "which is a display"},
		itemDirectory:   {tierSource(passingImport, "_, _ = os.Getwd()"), statementSlotLine, "calls os.Getwd"},
		itemEnvironment: {tierSource(passingImport, "os.Setenv(\"GOFLAGS\", \"\")"), statementSlotLine, "calls os.Setenv"},
		itemAbsolute:    {tierSource(passingImport, "_ = "+strconv.Quote(posix)), statementSlotLine, "absolute path literal"},
	}
}

const testModule = "github.com/iderex/gegenprobe"

func scanFixture(t *testing.T, src string, on map[string]bool) []violation {
	t.Helper()
	vs, scanned, err := scanTestSource("x_test.go", []byte(src), testModule, on)
	if err != nil {
		t.Fatalf("the fixture did not parse: %v", err)
	}
	if !scanned {
		t.Fatal("the fixture was taken for another tier and not read")
	}
	return vs
}

func TestAGateTestThatNeedsNothingPasses(t *testing.T) {
	if vs := scanFixture(t, passingTierSource(), enabled(items())); len(vs) != 0 {
		t.Fatalf("the passing fixture was refused: %v", vs)
	}
}

// The near miss, per item: one line different from a fixture that passes, the
// line somebody actually writes, and it has to trip that item and no other.
func TestEveryItemHasAFixtureThatTripsItAndNothingElse(t *testing.T) {
	for item, f := range capabilityFixtures(t) {
		t.Run(item, func(t *testing.T) {
			vs := scanFixture(t, f.src, enabled(items()))

			if len(vs) != 1 {
				t.Fatalf("the fixture for %s tripped %d items, %v; a fixture tripping two proves neither", item, len(vs), vs)
			}
			if vs[0].item != item {
				t.Fatalf("the fixture for %s tripped %s instead: %s", item, vs[0].item, vs[0].why)
			}
			if vs[0].line != f.line {
				t.Errorf("the refusal names line %d; the fixture changes line %d", vs[0].line, f.line)
			}
			if !strings.Contains(vs[0].why, f.says) {
				t.Errorf("the refusal does not say why in the terms the record uses: %q", vs[0].why)
			}
		})
	}
}

func TestEachFixtureIsOneLineFromThePassingOne(t *testing.T) {
	good := strings.Split(passingTierSource(), "\n")
	for item, f := range capabilityFixtures(t) {
		t.Run(item, func(t *testing.T) {
			bad := strings.Split(f.src, "\n")
			if len(good) != len(bad) {
				t.Fatalf("the fixtures differ in length, %d lines against %d", len(good), len(bad))
			}
			differing := 0
			for i := range good {
				if good[i] != bad[i] {
					differing++
				}
			}
			if differing != 1 {
				t.Fatalf("the fixture for %s differs on %d lines; a near miss that could not have passed proves less", item, differing)
			}
		})
	}
}

// Turning one item off has to turn its own fixture green and leave the others
// where they were. A fixture that is green with the item disabled is a fixture
// the item refused, rather than one nothing was ever asked about.
func TestDisablingAnItemTurnsItsFixtureGreen(t *testing.T) {
	for item, f := range capabilityFixtures(t) {
		t.Run(item, func(t *testing.T) {
			on := enabled(items())
			delete(on, item)

			if vs := scanFixture(t, f.src, on); len(vs) != 0 {
				t.Fatalf("with %s disabled its fixture is still refused: %v", item, vs)
			}
			if vs := scanFixture(t, f.src, enabled(items())); len(vs) != 1 {
				t.Fatalf("with every item enabled the same fixture reports %d violations", len(vs))
			}
		})
	}
}

func TestWithEveryItemDisabledEveryFixtureIsGreen(t *testing.T) {
	off := map[string]bool{}
	for item, f := range capabilityFixtures(t) {
		if vs := scanFixture(t, f.src, off); len(vs) != 0 {
			t.Errorf("the fixture for %s is refused by a check that judges nothing: %v", item, vs)
		}
	}
}

func TestTheFixturesCoverTheRefusableListExactly(t *testing.T) {
	f := capabilityFixtures(t)
	seen := map[string]bool{}
	for _, item := range items() {
		if _, ok := f[item]; !ok {
			t.Errorf("%s is on the refusable list with no fixture that trips it, so nothing proves it bites", item)
		}
		if seen[item] {
			t.Errorf("%s is on the refusable list twice", item)
		}
		seen[item] = true
	}
	for item := range f {
		if !seen[item] {
			t.Errorf("there is a fixture for %s, which is not on the refusable list", item)
		}
	}
}

// The second tier is exempt by construction rather than by exception. The
// source here trips two items and is not read at all, because its build
// constraint names another tier. The constraint carries its own line ending so
// that the literal holding it is not itself a candidate path.
func TestAFileInAnotherTierIsNotRead(t *testing.T) {
	posix := absolutePaths(t)[0]
	for _, c := range []struct{ tier, constraint string }{
		{"integration", "//go:build integration\n"},
		{"regression", "//go:build regression\n"},
	} {
		t.Run(c.tier, func(t *testing.T) {
			src := c.constraint + "\n" + tierSource("\"os/exec\"", "_ = "+strconv.Quote(posix))

			vs, scanned, err := scanTestSource("x_test.go", []byte(src), testModule, enabled(items()))
			if err != nil {
				t.Fatal(err)
			}
			if scanned {
				t.Fatalf("a %s tagged file was read as a gate test and reported %v", c.tier, vs)
			}
		})
	}
}

// moduleTree writes a module rooted tree holding the named test files, so the leg can
// walk something shaped like this repository.
func moduleTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+testModule+"\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
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

func TestTheLegNamesTheFileTheLineAndTheItem(t *testing.T) {
	root := moduleTree(t, map[string]string{"reader/reader_test.go": capabilityFixtures(t)[itemExec].src})

	o := capabilitiesLeg(root)

	if o.verdict != failed {
		t.Fatalf("a gate test importing os/exec passed the leg: %v %s", o.verdict, o.detail)
	}
	for _, want := range []string{"reader/reader_test.go:5", "imports os/exec", "(exec)", "#83"} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the failure does not carry %q:\n%s", want, o.detail)
		}
	}
}

func TestTheLegPassesATreeWhoseGateTestsNeedNothing(t *testing.T) {
	root := moduleTree(t, map[string]string{
		"reader/reader_test.go": passingTierSource(),
		"gate/gate_test.go":     passingTierSource(),
	})

	if o := capabilitiesLeg(root); o.verdict != passed {
		t.Fatalf("a clean tree was refused: %v %s", o.verdict, o.detail)
	}
}

// A tree the leg read nothing in reports a skip rather than a pass. A pass here
// would be a green line standing for no examination at all, which is the shape
// the whole command exists to refuse.
func TestTheLegSkipsWhereThereIsNoGateTestToRead(t *testing.T) {
	root := moduleTree(t, map[string]string{
		"harness/harness_test.go": "//go:build integration\n\n" + passingTierSource(),
	})

	o := capabilitiesLeg(root)

	if o.verdict != skipped {
		t.Fatalf("a tree holding only another tier did not report a skip: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "nothing was read") {
		t.Errorf("the skip does not say what was not examined: %q", o.detail)
	}
}

// Source under testdata is not compiled by the go tool, so it is not a test in
// any tier. The leg has to skip it, otherwise a fixture that exists to be bad
// fails the tree that holds it.
func TestSourceUnderTestdataIsNotRead(t *testing.T) {
	root := moduleTree(t, map[string]string{
		"reader/testdata/bad_test.go": capabilityFixtures(t)[itemExec].src,
		"reader/reader_test.go":       passingTierSource(),
	})

	if o := capabilitiesLeg(root); o.verdict != passed {
		t.Fatalf("source under testdata was judged: %v %s", o.verdict, o.detail)
	}
}

// A path is one line long. A test holding another file's source as a literal is
// how a fixture is written here, and such a literal opens with a build
// constraint comment often enough that the narrowing is what stops this check
// from refusing its own suite.
func TestAMultiLineLiteralIsNotAPath(t *testing.T) {
	for _, p := range absolutePaths(t) {
		if !isAbsolutePathLiteral(p) {
			t.Errorf("the absolute path %q was not recognised as one", p)
		}
	}
	if isAbsolutePathLiteral(passingTierSource()) {
		t.Error("a whole source file held as a literal was taken for a path")
	}
	if isAbsolutePathLiteral("testdata/reader.out") {
		t.Error("a relative path was taken for an absolute one")
	}
	if isAbsolutePathLiteral("") {
		t.Error("the empty string was taken for a path")
	}
}

func TestTheLegSaysWhereItCannotFindAModule(t *testing.T) {
	if o := capabilitiesLeg(t.TempDir()); o.verdict != failed {
		t.Fatalf("a tree with no go.mod did not fail: %v %s", o.verdict, o.detail)
	}
}
