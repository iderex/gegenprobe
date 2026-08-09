package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// knownLegs is the leg set the fixtures are judged against. It is stated here
// rather than taken from legs(), so a row in a fixture is refused for what the
// fixture says and not for what this repository's run list happened to hold on
// the day the test ran.
func knownLegs() map[string]bool {
	return map[string]bool{"tests": true, "dependencies": true}
}

// parityDocument builds a whole document around one table row. Every case below
// substitutes exactly that row, so a verdict that moves between two of them
// moved for the row and not for anything around it.
func parityDocument(required, row string) string {
	return "# Quality parity\n" +
		"\n" +
		"Target list taken: 2026-08-09\n" +
		"Required status checks: " + required + "\n" +
		"\n" +
		"The workflow in this tree is `.github/workflows/ci.yml`.\n" +
		"\n" +
		"| " + headerCell + " | What it covers there | Covered here by | Blocks a merge here | Issue | Note |\n" +
		"| --- | --- | --- | --- | --- | --- |\n" +
		row + "\n"
}

const passingRow = "| build | that the tree compiles and its unit tests pass | leg tests; workflow .github/workflows/ci.yml | no | #68 | - |"

// parityTree writes the document into a tree holding one workflow file, which
// is what the leg reads besides the document itself.
func parityTree(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".github", "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".github", "workflows", "ci.yml"), []byte("name: CI\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(parityDoc)), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestParityAcceptsARowThatNamesWhatCoversItHere(t *testing.T) {
	o := judgeParity(parityTree(t, parityDocument("none", passingRow)), knownLegs())

	if o.verdict != passed {
		t.Fatalf("a row naming a leg and a workflow that both exist was refused: %s", o.detail)
	}
	if !strings.Contains(o.detail, "1 row(s) read against the target list taken 2026-08-09") {
		t.Errorf("the pass does not say what it read: %q", o.detail)
	}
}

// The refusal the issue asks for by name. This fixture differs from the passing
// one by exactly the row, and inside the row by the two cells that carry the
// whole obligation: what covers the check here, and why nothing does.
func TestParityRefusesARowWithNeitherACoveringCheckNorAReason(t *testing.T) {
	row := "| build | that the tree compiles and its unit tests pass | - | no | #68 | - |"

	o := judgeParity(parityTree(t, parityDocument("none", row)), knownLegs())

	if o.verdict != failed {
		t.Fatalf("a row with a blank covering cell and no reason passed: %v %s", o.verdict, o.detail)
	}
	for _, want := range []string{"quality-parity.md:10", "`build`", "names nothing here that covers it and gives no reason"} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the failure does not carry %q:\n%s", want, o.detail)
		}
	}
}

// A deviation is a row with no cover and a reason, which is the shape most of
// the real table is in. It has to pass, or the refusal above would be a rule
// against deviating rather than against deviating in silence.
func TestParityAcceptsADeviationThatCarriesItsReason(t *testing.T) {
	row := "| prettier | formatting of the web assets | - | no | - | this board has no web assets |"

	if o := judgeParity(parityTree(t, parityDocument("none", row)), knownLegs()); o.verdict != passed {
		t.Fatalf("a deviation carrying its reason was refused: %s", o.detail)
	}
}

func TestTheTwoFixturesDifferByExactlyOneCell(t *testing.T) {
	blank := "| build | that the tree compiles and its unit tests pass | - | no | #68 | - |"

	good := splitCells(passingRow)
	bad := splitCells(blank)

	if len(good) != len(bad) {
		t.Fatalf("the fixtures hold %d cells against %d", len(good), len(bad))
	}
	differing := 0
	for i := range good {
		if good[i] != bad[i] {
			differing++
		}
	}
	if differing != 1 {
		t.Fatalf("the fixtures differ in %d cells; the near miss proves less than one cell apart would", differing)
	}
}

// Every other way a row can name something that is not there. Each is one cell
// away from the passing row.
func TestParityRefusesARowThatNamesSomethingThisTreeDoesNotHold(t *testing.T) {
	for _, c := range []struct{ name, row, want string }{
		{
			"a leg that does not exist",
			"| build | that the tree compiles | leg tets | no | #68 | - |",
			"`tets` is named as a leg of this command and there is no such leg",
		},
		{
			"a workflow that does not exist",
			"| build | that the tree compiles | workflow .github/workflows/cli.yml | no | #68 | - |",
			"is named as a workflow in this tree and there is no such file",
		},
		{
			"an entry in neither form",
			"| build | that the tree compiles | the gate | no | #68 | - |",
			"is neither `leg <name>` nor `workflow <path>`",
		},
		{
			"nothing said about what the check covers there",
			"| build | - | leg tests | no | #68 | - |",
			"does not say what the check covers there",
		},
		{
			"an issue reference that is not one",
			"| build | that the tree compiles | leg tests | no | 68 | - |",
			"in the issue column, which is not #NNN",
		},
		{
			"a merge column that is neither yes nor no",
			"| build | that the tree compiles | leg tests | maybe | #68 | - |",
			"which is neither yes nor no",
		},
		{
			"a column dropped",
			"| build | that the tree compiles | leg tests | no | #68 |",
			"5 cells rather than 6",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := judgeParity(parityTree(t, parityDocument("none", c.row)), knownLegs())
			if o.verdict != failed {
				t.Fatalf("the row passed: %v %s", o.verdict, o.detail)
			}
			if !strings.Contains(o.detail, c.want) {
				t.Errorf("the failure does not say why:\n%s", o.detail)
			}
		})
	}
}

// The merge column fails in the safe direction. A row claiming a check blocks a
// merge is refused while the declaration at the top of the document says none
// does, so the table cannot claim a force the ruleset was never given.
func TestParityRefusesAMergeClaimTheDeclarationDoesNotCarry(t *testing.T) {
	row := "| build | that the tree compiles | leg tests | yes | #68 | - |"

	o := judgeParity(parityTree(t, parityDocument("none", row)), knownLegs())

	if o.verdict != failed {
		t.Fatalf("a row claiming to block a merge passed against a declaration of none: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "Required status checks field on line 4 does not name it") {
		t.Errorf("the failure does not point at the declaration:\n%s", o.detail)
	}
}

func TestParityAcceptsAMergeClaimTheDeclarationCarries(t *testing.T) {
	row := "| build | that the tree compiles | leg tests | yes | #68 | - |"

	if o := judgeParity(parityTree(t, parityDocument("build", row)), knownLegs()); o.verdict != passed {
		t.Fatalf("a declared required check was refused in the merge column: %s", o.detail)
	}
}

// The other direction the leg can decide without reading a matrix: a workflow
// arrives here and no row says what on the target board it answers.
func TestParityRefusesAWorkflowNoRowMentions(t *testing.T) {
	root := parityTree(t, parityDocument("none", passingRow))
	if err := os.WriteFile(filepath.Join(root, ".github", "workflows", "nightly.yml"), []byte("name: Nightly\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	o := judgeParity(root, knownLegs())

	if o.verdict != failed {
		t.Fatalf("a workflow the table names nowhere passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, ".github/workflows/nightly.yml is a workflow in this tree that the table names nowhere") {
		t.Errorf("the failure does not name the workflow:\n%s", o.detail)
	}
}

func TestParityRefusesADocumentMissingItsFieldsOrItsTable(t *testing.T) {
	for _, c := range []struct{ name, body, want string }{
		{
			"no date the target list was taken",
			strings.Replace(parityDocument("none", passingRow), "Target list taken: 2026-08-09\n", "", 1),
			"makes no claim about when it was derived",
		},
		{
			"no declaration of what blocks a merge",
			strings.Replace(parityDocument("none", passingRow), "Required status checks: none\n", "", 1),
			"the merge column is checked against nothing",
		},
		{
			"no table at all",
			"# Quality parity\n\nTarget list taken: 2026-08-09\nRequired status checks: none\n\n`.github/workflows/ci.yml`\n",
			"no table whose first column is " + headerCell,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := judgeParity(parityTree(t, c.body), knownLegs())
			if o.verdict != failed {
				t.Fatalf("the document passed: %v %s", o.verdict, o.detail)
			}
			if !strings.Contains(o.detail, c.want) {
				t.Errorf("the failure does not say what was missing:\n%s", o.detail)
			}
		})
	}
}

// A tree with no table is a failure rather than a skip. The skip verdict is for
// a check that could not examine its subject, and here the subject is a file
// this repository owes rather than one that may or may not be present.
func TestParityRefusesATreeWithNoTableAtAll(t *testing.T) {
	o := judgeParity(t.TempDir(), knownLegs())

	if o.verdict != failed {
		t.Fatalf("a tree with no parity document did not fail: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "no check on the target board is mapped onto this one") {
		t.Errorf("the failure does not say what is absent: %q", o.detail)
	}
}

// The table this repository ships is judged by the same code, against the leg
// names the run list actually holds. A leg renamed in gate/legs.go and not in
// the document reddens here before it reddens a run.
func TestTheTableThisRepositoryShipsPasses(t *testing.T) {
	o := parityLeg("..")

	if o.verdict != passed {
		t.Fatalf("the parity table in this tree is refused by its own leg:\n%s", o.detail)
	}
	if !strings.Contains(o.detail, "row(s) read against the target list taken") {
		t.Errorf("the pass says nothing about what it read: %q", o.detail)
	}
}

// Disabling the leg turns the refusing fixtures green, which is what says the
// verdicts above came from this leg rather than from something else the tree
// was doing.
func TestWithoutTheLegNothingRefusesAnyOfTheFixtures(t *testing.T) {
	for _, row := range []string{
		"| build | that the tree compiles | - | no | #68 | - |",
		"| build | that the tree compiles | leg tets | no | #68 | - |",
		"| build | that the tree compiles | leg tests | yes | #68 | - |",
	} {
		root := parityTree(t, parityDocument("none", row))
		for _, l := range legs() {
			if l.name == "quality parity" {
				continue
			}
			if l.name == "documentation form" || l.name == "documented paths" || l.name == "documentation links" {
				if o := l.run(root); o.verdict == failed {
					t.Errorf("leg %q refused a fixture the quality parity leg is meant to judge:\n%s", l.name, o.detail)
				}
			}
		}
	}
}
