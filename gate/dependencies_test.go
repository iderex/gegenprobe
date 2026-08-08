package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The passing pair every case below changes one line of: a module file with one
// direct requirement, and a document declaring one and giving it a reason.
func goodGoMod() string {
	return strings.Join([]string{
		"module github.com/iderex/gegenprobe",
		"",
		"go 1.24",
		"",
		"require (",
		"\tgithub.com/example/reader v1.2.3",
		"\tgithub.com/example/plumbing v0.4.0 // indirect",
		")",
	}, "\n")
}

func goodBudget() string {
	return strings.Join([]string{
		"# The dependency budget",
		"",
		"Direct dependencies: 1",
		"",
		"## github.com/example/reader",
		"",
		"Reads a fixed format table that the standard library does not.",
	}, "\n")
}

func TestThePassingPairAgrees(t *testing.T) {
	if o := judgeBudget([]byte(goodGoMod()), []byte(goodBudget())); o.verdict != passed {
		t.Fatalf("a module file agreeing with its budget was refused: %s", o.detail)
	}
}

// The near miss, and the one a count would get wrong. An indirect requirement is
// not a dependency this repository chose, so it is neither in the budget nor
// owed a reason.
func TestAnIndirectRequirementIsNotInTheBudget(t *testing.T) {
	direct := directRequires([]byte(goodGoMod()))

	if len(direct) != 1 || direct[0] != "github.com/example/reader" {
		t.Fatalf("direct requirements = %v, want only github.com/example/reader", direct)
	}
}

func TestASingleLineRequireIsCounted(t *testing.T) {
	gomod := "module m\n\ngo 1.24\n\nrequire github.com/example/reader v1.2.3\n"

	if direct := directRequires([]byte(gomod)); len(direct) != 1 || direct[0] != "github.com/example/reader" {
		t.Fatalf("direct requirements = %v, want github.com/example/reader", direct)
	}
}

// The failure this leg exists for: a dependency arrived and nothing says why.
func TestADependencyWithNoReasonIsRefused(t *testing.T) {
	budget := strings.Replace(goodBudget(), "## github.com/example/reader", "## github.com/example/other", 1)
	budget = strings.Replace(budget, "Direct dependencies: 1", "Direct dependencies: 1", 1)

	o := judgeBudget([]byte(goodGoMod()), []byte(budget))

	if o.verdict != failed {
		t.Fatalf("a dependency with no recorded reason passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "github.com/example/reader") {
		t.Errorf("the failure does not name the dependency:\n%s", o.detail)
	}
}

// The other direction, which is how a register stops describing the tree.
func TestAReasonForSomethingNotRequiredIsRefused(t *testing.T) {
	gomod := strings.Replace(goodGoMod(), "\tgithub.com/example/reader v1.2.3\n", "", 1)
	budget := strings.Replace(goodBudget(), "Direct dependencies: 1", "Direct dependencies: 0", 1)

	o := judgeBudget([]byte(gomod), []byte(budget))

	if o.verdict != failed {
		t.Fatalf("a reason for a module nothing requires passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "github.com/example/reader") {
		t.Errorf("the failure does not name the stale entry:\n%s", o.detail)
	}
}

func TestTheNumberHasToMatchWhatIsRequired(t *testing.T) {
	budget := strings.Replace(goodBudget(), "Direct dependencies: 1", "Direct dependencies: 2", 1)

	o := judgeBudget([]byte(goodGoMod()), []byte(budget))

	if o.verdict != failed {
		t.Fatalf("a budget disagreeing with go.mod passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "budget of 2") {
		t.Errorf("the failure does not name the declared number:\n%s", o.detail)
	}
}

// A budget document that stops declaring a budget is a failure rather than a
// budget of zero, which is the value under which a tree with dependencies would
// be refused for the wrong reason and one without them would pass for no reason.
func TestEachWayTheBudgetGoesMissingIsRefused(t *testing.T) {
	for _, c := range []struct{ name, budget, says string }{
		{"no number", strings.Replace(goodBudget(), "Direct dependencies: 1", "", 1), "no `Direct dependencies:` number"},
		{"a number that is not one", strings.Replace(goodBudget(), "Direct dependencies: 1", "Direct dependencies: a few", 1), "no `Direct dependencies:` number"},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := judgeBudget([]byte(goodGoMod()), []byte(c.budget))

			if o.verdict != failed {
				t.Fatalf("%s was accepted", c.name)
			}
			if !strings.Contains(o.detail, c.says) {
				t.Errorf("the refusal does not say what is missing:\n%s", o.detail)
			}
		})
	}
}

// The verdict is on the exit status rather than on the wording, so this fixture
// carries no sentence the toolchain has to keep printing.
func TestAFailedChecksumVerificationIsRefused(t *testing.T) {
	o := judgeVerify("github.com/example/reader v1.2.3: dir has been modified\n", errors.New("exit status 1"))

	if o.verdict != failed {
		t.Fatalf("a non zero verification passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "dir has been modified") {
		t.Errorf("the failure drops what the tool said:\n%s", o.detail)
	}
	if o := judgeVerify("all modules verified\n", nil); o.verdict != passed {
		t.Errorf("a successful verification was refused: %s", o.detail)
	}
}

// Drift in either file, and the case that would otherwise be invisible: a build
// creating a go.sum where the tree had none.
func TestABuildThatRewritesAModuleFileIsRefused(t *testing.T) {
	before := moduleFiles{gomod: []byte("module m\n\ngo 1.24\n")}

	for _, c := range []struct {
		name  string
		after moduleFiles
		says  string
	}{
		{"go.mod rewritten", moduleFiles{gomod: []byte("module m\n\ngo 1.24\n\nrequire x v1\n")}, "go.mod"},
		{"go.sum created", moduleFiles{gomod: before.gomod, gosum: []byte("x v1 h1:...\n")}, "go.sum"},
		{"both", moduleFiles{gomod: []byte("module m\n"), gosum: []byte("x v1 h1:...\n")}, "go.mod and go.sum"},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := judgeDrift(before, c.after)

			if o.verdict != failed {
				t.Fatalf("%s passed", c.name)
			}
			if !strings.Contains(o.detail, c.says) {
				t.Errorf("the failure does not name what moved:\n%s", o.detail)
			}
		})
	}

	if o := judgeDrift(before, before); o.verdict != passed {
		t.Errorf("an unchanged pair of module files was refused: %s", o.detail)
	}
}

// The budget this repository actually declares has to agree with the module file
// it is about, or the leg fails for a reason nobody changed.
func TestTheBudgetInThisTreeAgreesWithGoMod(t *testing.T) {
	gomod, err := os.ReadFile(filepath.Join("..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(dependencyDoc)))
	if err != nil {
		t.Fatal(err)
	}
	if o := judgeBudget(gomod, doc); o.verdict != passed {
		t.Fatalf("this tree's budget and go.mod disagree: %s", o.detail)
	}
}

// The leg reads two files before it reaches for a toolchain, and each of them
// missing is a failure rather than a pass over nothing.
func TestTheLegRefusesATreeWithNothingToRead(t *testing.T) {
	root := t.TempDir()

	o := dependenciesLeg(root)
	if o.verdict != failed || !strings.Contains(o.detail, "go.mod") {
		t.Fatalf("a tree with no go.mod reported %v: %s", o.verdict, o.detail)
	}

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goodGoMod()), 0o644); err != nil {
		t.Fatal(err)
	}
	o = dependenciesLeg(root)
	if o.verdict != failed || !strings.Contains(o.detail, "dependency budget") {
		t.Fatalf("a tree with no %s reported %v: %s", dependencyDoc, o.verdict, o.detail)
	}
}

// A budget the leg refuses is refused before anything is run, so the verdict
// does not depend on a toolchain being present.
func TestTheLegRefusesADisagreeingBudgetBeforeItRunsAnything(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goodGoMod()), 0o644); err != nil {
		t.Fatal(err)
	}
	budget := strings.Replace(goodBudget(), "Direct dependencies: 1", "Direct dependencies: 3", 1)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(dependencyDoc)), []byte(budget), 0o644); err != nil {
		t.Fatal(err)
	}

	o := dependenciesLeg(root)

	if o.verdict != failed {
		t.Fatalf("a budget disagreeing with go.mod reported %v: %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "budget of 3") {
		t.Errorf("the failure does not name the declared number:\n%s", o.detail)
	}
}

// legsThatJudgeTheBudget is the run with and without this leg.
func legsThatJudgeTheBudget(withDependencies bool, gomod, doc string) []leg {
	ls := []leg{
		{name: "format", subject: "fixture", run: formatLeg},
	}
	if withDependencies {
		ls = append(ls, leg{
			name:    "dependencies",
			subject: "fixture",
			run:     func(string) outcome { return judgeBudget([]byte(gomod), []byte(doc)) },
		})
	}
	return ls
}

func TestDisablingTheDependenciesLegTurnsItsFixturesGreen(t *testing.T) {
	for _, c := range []struct{ name, gomod, doc string }{
		{"no reason", goodGoMod(), strings.Replace(goodBudget(), "## github.com/example/reader", "## github.com/example/other", 1)},
		{"the wrong number", goodGoMod(), strings.Replace(goodBudget(), "Direct dependencies: 1", "Direct dependencies: 2", 1)},
		{"no budget at all", goodGoMod(), strings.Replace(goodBudget(), "Direct dependencies: 1", "", 1)},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()

			if status := run(io.Discard, root, legsThatJudgeTheBudget(true, c.gomod, c.doc)); status == 0 {
				t.Fatalf("the gate passed %s with the leg enabled", c.name)
			}
			if status := run(io.Discard, root, legsThatJudgeTheBudget(false, c.gomod, c.doc)); status != 0 {
				t.Errorf("the gate still refused %s with the leg disabled, so the fixture proves something else", c.name)
			}
		})
	}
}
