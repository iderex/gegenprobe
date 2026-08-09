package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// registerDocument builds a whole document around one recorded output and one
// table row. Every case below substitutes exactly one of the two, so a verdict
// that moves between two of them moved for what was substituted and not for
// anything around it.
func registerDocument(recorded, row string) string {
	return "# The supply chain score\n" +
		"\n" +
		"Output taken: 2026-08-09\n" +
		"Scorecard version: v5.5.0\n" +
		"Scored commit: 772e86784f782d945c0702b1fd20699f8473c024\n" +
		"\n" +
		outputHeading + "\n" +
		"\n" +
		"```\n" +
		recorded + "\n" +
		"```\n" +
		"\n" +
		"| " + registerHeaderCell + " | Score | Outcome | What retires the acceptance | Issue | Reason |\n" +
		"| --- | --- | --- | --- | --- | --- |\n" +
		row + "\n"
}

const recordedOne = "Fuzzing 0"

// acceptedRow is the row every case below is one cell away from.
const acceptedRow = "| Fuzzing | 0 | accepted | #61 lands a fuzz target per reader | #61 | the surface worth fuzzing is the readers and none is written |"

func registerTree(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(registerDoc)), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestTheRegisterAcceptsACheckAnsweredOnce(t *testing.T) {
	o := supplyChainLeg(registerTree(t, registerDocument(recordedOne, acceptedRow)))

	if o.verdict != passed {
		t.Fatalf("a check with one row answering it was refused: %s", o.detail)
	}
	for _, want := range []string{
		"1 check(s) answered against the output taken 2026-08-09 from Scorecard v5.5.0",
		"0 fixed, 1 accepted with what retires each, 0 not applicable",
		"no leg here reads a live score",
	} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the pass does not say what it read, wanted %q:\n%s", want, o.detail)
		}
	}
}

// The first refusal the issue asks for by name. This fixture differs from the
// passing one by the recorded output holding one more check, and by nothing
// else, so what is refused is the absence of the row rather than anything about
// the row that is there.
func TestTheRegisterRefusesACheckInTheOutputThatNoRowAnswers(t *testing.T) {
	o := supplyChainLeg(registerTree(t, registerDocument(recordedOne+"\nSecurity-Policy 0", acceptedRow)))

	if o.verdict != failed {
		t.Fatalf("a check in the recorded output with no row passed: %v %s", o.verdict, o.detail)
	}
	for _, want := range []string{"supply-chain.md:1", "`Security-Policy`", "no row answers it"} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the failure does not carry %q:\n%s", want, o.detail)
		}
	}
}

// The second refusal the issue asks for by name, and the same fixture from the
// other end: the row is the passing one and the recorded output no longer names
// the check it answers.
func TestTheRegisterRefusesARowAnsweringACheckTheOutputDoesNotName(t *testing.T) {
	o := supplyChainLeg(registerTree(t, registerDocument("Security-Policy 0", acceptedRow)))

	if o.verdict != failed {
		t.Fatalf("a row answering a check the output does not name passed: %v %s", o.verdict, o.detail)
	}
	for _, want := range []string{"`Fuzzing`", "answers a check the recorded output does not name"} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the failure does not carry %q:\n%s", want, o.detail)
		}
	}
}

// An acceptance with no condition that retires it is the shape this register
// exists against, and it is one cell away from the row that passes.
func TestTheRegisterRefusesAnAcceptanceThatNamesNothingRetiringIt(t *testing.T) {
	row := "| Fuzzing | 0 | accepted | - | #61 | the surface worth fuzzing is the readers and none is written |"

	o := supplyChainLeg(registerTree(t, registerDocument(recordedOne, row)))

	if o.verdict != failed {
		t.Fatalf("an acceptance carrying no retirement condition passed: %v %s", o.verdict, o.detail)
	}
	for _, want := range []string{"`Fuzzing`", "makes it a dispensation rather than a debt"} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the failure does not carry %q:\n%s", want, o.detail)
		}
	}
}

func TestTheAcceptedAndUnretiredRowsDifferByExactlyOneCell(t *testing.T) {
	unretired := "| Fuzzing | 0 | accepted | - | #61 | the surface worth fuzzing is the readers and none is written |"

	good := splitCells(acceptedRow)
	bad := splitCells(unretired)

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

// Every other way a row can fail to answer the check it names. Each is one cell
// away from a row that passes.
func TestTheRegisterRefusesARowThatAnswersNothing(t *testing.T) {
	for _, c := range []struct{ name, recorded, row, want string }{
		{
			"a score that disagrees with the recording",
			"Fuzzing 7",
			acceptedRow,
			"says `0` where the recorded output says `7`",
		},
		{
			"an outcome that is none of the three",
			recordedOne,
			"| Fuzzing | 0 | triaged | #61 lands a fuzz target per reader | #61 | none is written |",
			"says `triaged` in the outcome column",
		},
		{
			"a fixed row carrying a retirement condition",
			"Binary-Artifacts 10",
			"| Binary-Artifacts | 10 | fixed | somebody commits a binary | - | no binary is committed to this tree |",
			"is `fixed` and names something that would retire it",
		},
		{
			"a row with no reason",
			recordedOne,
			"| Fuzzing | 0 | accepted | #61 lands a fuzz target per reader | #61 | - |",
			"gives no reason",
		},
		{
			"an issue reference that is not a number",
			recordedOne,
			"| Fuzzing | 0 | accepted | #61 lands a fuzz target per reader | the fuzzing issue | none is written |",
			"which is not #NNN",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := supplyChainLeg(registerTree(t, registerDocument(c.recorded, c.row)))

			if o.verdict != failed {
				t.Fatalf("%s passed: %v %s", c.name, o.verdict, o.detail)
			}
			if !strings.Contains(o.detail, c.want) {
				t.Errorf("the failure does not carry %q:\n%s", c.want, o.detail)
			}
		})
	}
}

// A check answered twice is two answers to one question, and the second is the
// one nobody reads.
func TestTheRegisterRefusesASecondRowForOneCheck(t *testing.T) {
	o := supplyChainLeg(registerTree(t, registerDocument(recordedOne, acceptedRow+"\n"+acceptedRow)))

	if o.verdict != failed {
		t.Fatalf("two rows answering one check passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "a second row answers `Fuzzing`") {
		t.Errorf("the failure does not name the second row:\n%s", o.detail)
	}
}

// A fixed row and a not applicable row both pass, or the refusals above would be
// rules against answering rather than against answering with nothing.
func TestTheRegisterAcceptsTheOtherTwoOutcomes(t *testing.T) {
	for _, c := range []struct{ name, recorded, row string }{
		{
			"fixed",
			"Binary-Artifacts 10",
			"| Binary-Artifacts | 10 | fixed | - | - | no binary is committed to this tree |",
		},
		{
			"not applicable, at the score the tool uses for a check it did not apply",
			"Signed-Releases -1",
			"| Signed-Releases | -1 | not applicable | - | #63; #75 | there is no release for the check to look at |",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if o := supplyChainLeg(registerTree(t, registerDocument(c.recorded, c.row))); o.verdict != passed {
				t.Fatalf("a %s row was refused: %s", c.name, o.detail)
			}
		})
	}
}

// The three fields the leg reads, each removed from a document that otherwise
// passes.
func TestTheRegisterRefusesADocumentMissingAFieldTheLegReads(t *testing.T) {
	for _, c := range []struct{ name, line, want string }{
		{"when the output was recorded", "Output taken: 2026-08-09", "no `Output taken: YYYY-MM-DD` field"},
		{"which version produced it", "Scorecard version: v5.5.0", "no `Scorecard version:` field"},
		{"which commit was scored", "Scored commit: 772e86784f782d945c0702b1fd20699f8473c024", "no `Scored commit:` field"},
	} {
		t.Run(c.name, func(t *testing.T) {
			body := strings.Replace(registerDocument(recordedOne, acceptedRow), c.line+"\n", "", 1)

			o := supplyChainLeg(registerTree(t, body))

			if o.verdict != failed {
				t.Fatalf("a document with no %s passed: %v %s", c.name, o.verdict, o.detail)
			}
			if !strings.Contains(o.detail, c.want) {
				t.Errorf("the failure does not carry %q:\n%s", c.want, o.detail)
			}
		})
	}
}

// A document with no recorded output is a register checked against nothing, and
// it has to be refused rather than passing with no rows to compare.
func TestTheRegisterRefusesADocumentWithNoRecordedOutput(t *testing.T) {
	body := strings.Replace(registerDocument(recordedOne, acceptedRow), outputHeading, "## Something else", 1)

	o := supplyChainLeg(registerTree(t, body))

	if o.verdict != failed {
		t.Fatalf("a register with no recorded output passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "is checked against no output at all") {
		t.Errorf("the failure does not say the register was checked against nothing:\n%s", o.detail)
	}
}

// A line in the recorded block that is neither a check nor a score is reported
// rather than skipped, because a block the leg reads half of is a register
// checked against half an output.
func TestTheRegisterRefusesALineInTheOutputThatIsNotACheckAndAScore(t *testing.T) {
	o := supplyChainLeg(registerTree(t, registerDocument(recordedOne+"\nSecurity-Policy: 0", acceptedRow)))

	if o.verdict != failed {
		t.Fatalf("an unreadable line in the recorded output passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "which is not a check name and a score") {
		t.Errorf("the failure does not name the unreadable line:\n%s", o.detail)
	}
}

func TestTheRegisterRefusesADocumentThatIsNotInTheTree(t *testing.T) {
	o := supplyChainLeg(t.TempDir())

	if o.verdict != failed {
		t.Fatalf("a tree with no register passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "has an outcome anywhere") {
		t.Errorf("the failure does not say what is missing:\n%s", o.detail)
	}
}

// A table with a column added or removed is a document the leg would otherwise
// read cell by cell into the wrong fields.
func TestTheRegisterRefusesARowWithTheWrongNumberOfCells(t *testing.T) {
	row := "| Fuzzing | 0 | accepted | #61 lands a fuzz target per reader | #61 |"

	o := supplyChainLeg(registerTree(t, registerDocument(recordedOne, row)))

	if o.verdict != failed {
		t.Fatalf("a row with five cells rather than six passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "5 cells rather than 6") {
		t.Errorf("the failure does not say how many cells it found:\n%s", o.detail)
	}
}
