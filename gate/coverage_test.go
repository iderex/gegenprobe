package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goodFloor is the document every case below changes one line of.
func goodFloorLines() []string {
	return []string{
		"# The coverage floor",
		"",
		"Floor: 82.2",
		"Last raised: 2026-08-08",
		"",
		"Prose about what the number does not cover.",
	}
}

func goodFloor() string { return strings.Join(goodFloorLines(), "\n") }

// withoutLine returns the document with the line at i replaced, which is what
// keeps each case one line away from a passing one.
func withoutLine(i int, replacement string) string {
	lines := append([]string(nil), goodFloorLines()...)
	if replacement == "" {
		return strings.Join(append(lines[:i:i], lines[i+1:]...), "\n")
	}
	lines[i] = replacement
	return strings.Join(lines, "\n")
}

func TestTheFloorDocumentIsRead(t *testing.T) {
	f, err := parseFloor([]byte(goodFloor()))
	if err != nil {
		t.Fatalf("a well formed floor document was refused: %v", err)
	}
	if f.percent != 82.2 {
		t.Errorf("floor = %v, want 82.2", f.percent)
	}
	if f.raised != "2026-08-08" {
		t.Errorf("last raised = %q, want 2026-08-08", f.raised)
	}
}

// Each way the document stops declaring a floor, one line from the passing one.
// A floor that cannot be read has to be a failure rather than a floor of zero:
// zero is the value under which every run is green while nothing is covered.
func TestEachWayTheFloorGoesMissingIsRefused(t *testing.T) {
	for _, c := range []struct {
		name, doc, says string
	}{
		{"no floor line", withoutLine(2, ""), "no `Floor:` line"},
		{"a floor that is not a number", withoutLine(2, "Floor: most of it"), "no `Floor:` line"},
		{"a number no float holds", withoutLine(2, "Floor: "+strings.Repeat("9", 400)), "not a number"},
		{"no date", withoutLine(3, ""), "no `Last raised:` date"},
		{"a date that is not one", withoutLine(3, "Last raised: recently"), "no `Last raised:` date"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := parseFloor([]byte(c.doc))
			if err == nil {
				t.Fatalf("%s was accepted", c.name)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the refusal does not say what is missing: %v", err)
			}
		})
	}
}

// The one thing this leg exists to do.
func TestADropBelowTheFloorFails(t *testing.T) {
	floor := floorRecord{percent: 82.2, raised: "2026-08-08"}

	o := judgeCoverage(82.1, floor)

	if o.verdict != failed {
		t.Fatalf("coverage below the floor reported %v: %s", o.verdict, o.detail)
	}
	for _, want := range []string{"82.1", "82.2", "2026-08-08"} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the failure does not carry %s, so a reader cannot see how far it fell:\n%s", want, o.detail)
		}
	}
}

// The near miss, which is the run that is exactly at the floor. A leg that
// refused it would be one nobody could ever satisfy on the day the floor was
// set from the measurement.
func TestCoverageExactlyAtTheFloorPasses(t *testing.T) {
	o := judgeCoverage(82.2, floorRecord{percent: 82.2, raised: "2026-08-08"})

	if o.verdict != passed {
		t.Fatalf("coverage exactly at the floor reported %v: %s", o.verdict, o.detail)
	}
}

// The caveat is the deliverable rather than a footnote, so the number cannot be
// printed without it. This asserts the pass and the failure both carry it, in
// the same place as the figure.
func TestTheNumberIsNeverPrintedWithoutWhatItDoesNotCover(t *testing.T) {
	floor := floorRecord{percent: 82.2, raised: "2026-08-08"}

	for _, o := range []outcome{judgeCoverage(90.0, floor), judgeCoverage(10.0, floor)} {
		if !strings.Contains(o.detail, "integration harness") || !strings.Contains(o.detail, floorDoc) {
			t.Errorf("a coverage verdict carried the number without what it does not cover:\n%s", o.detail)
		}
	}
}

func TestARunWellAboveTheFloorSaysSo(t *testing.T) {
	o := judgeCoverage(90.0, floorRecord{percent: 82.2, raised: "2026-08-08"})

	if o.verdict != passed {
		t.Fatalf("coverage above the floor reported %v: %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "Raising it") {
		t.Errorf("a total well above the floor does not suggest raising it:\n%s", o.detail)
	}
}

// The integration tier contributes nothing to the number because the run never
// asks for it. This asserts the invocation, which is the whole of how the figure
// could be inflated: a `-tags integration` run would count tests that start
// containers. What it does not do is watch two runs and compare them, and no
// gate tier test can, since record 0009 refuses os/exec here.
func TestTheCoverageRunAsksForNoTierButTheGate(t *testing.T) {
	args := coverageArgs("profile.out")

	want := []string{"test", "-coverprofile=profile.out", "./..."}
	if !equalStrings(args, want) {
		t.Fatalf("the coverage invocation is %v, want %v. A flag added here changes which tier is measured, so it is asserted rather than reviewed.", args, want)
	}
	for _, a := range args {
		for _, tier := range []string{"integration", "regression", "-tags"} {
			if strings.Contains(a, tier) {
				t.Errorf("the coverage run asks for %q, which measures a tier that is not the gate", a)
			}
		}
	}
}

// A listing with no total is a measurement that did not happen, and the verdict
// for that is a failure rather than a number.
func TestAListingWithNoTotalIsRefused(t *testing.T) {
	if _, err := totalFrom("github.com/x/y/main.go:12:\trender\t100.0%\n"); err == nil {
		t.Fatal("a listing with no total line was accepted")
	}
	total, err := totalFrom("github.com/x/y/main.go:12:\trender\t100.0%\ntotal:\t\t\t(statements)\t\t74.2%\n")
	if err != nil {
		t.Fatalf("a listing carrying a total was refused: %v", err)
	}
	if total != 74.2 {
		t.Errorf("total = %v, want 74.2", total)
	}
}

// The floor this repository actually declares has to be readable by the code
// that reads it, or the leg fails for a reason nobody changed.
func TestTheFloorDocumentInThisTreeParses(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(floorDoc)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseFloor(doc); err != nil {
		t.Fatalf("%s does not parse: %v", floorDoc, err)
	}
}

// legsThatJudgeCoverage is the run with and without this leg. A fixture still
// refused once the leg is gone is a fixture proving something else.
func legsThatJudgeCoverage(withCoverage bool, total float64, floor floorRecord) []leg {
	ls := []leg{
		{name: "format", subject: "fixture", run: formatLeg},
	}
	if withCoverage {
		ls = append(ls, leg{
			name:    "coverage",
			subject: "fixture",
			run:     func(string) outcome { return judgeCoverage(total, floor) },
		})
	}
	return ls
}

func TestDisablingTheCoverageLegTurnsItsFixtureGreen(t *testing.T) {
	root := t.TempDir()
	floor := floorRecord{percent: 82.2, raised: "2026-08-08"}

	if status := run(io.Discard, root, legsThatJudgeCoverage(true, 40.0, floor)); status == 0 {
		t.Fatal("the gate passed a total below the floor with the leg enabled")
	}
	if status := run(io.Discard, root, legsThatJudgeCoverage(false, 40.0, floor)); status != 0 {
		t.Error("the gate still refused with the leg disabled, so the fixture proves something else")
	}
}
