package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Coverage is the number most easily quoted as though it meant more than it
// does. What is measured here is the gate tier, and the parts of this project
// that most need testing are not in it: the runner against a real container
// engine and the recipes against real codes are the integration harness, under
// their own tag, and contribute nothing. So the caveat is printed in the same
// place as the figure rather than kept in a document beside it. A reader who
// quotes the number without it has to have skipped a sentence attached to it.
//
// The floor lives in the tree rather than in this source, in floorRecord below,
// because lowering it is the move somebody makes to turn a red run green and
// that should be a change to a document with a date on it.

// floorDoc is the file holding the floor and the date it was last raised. The
// same file carries the argument for both, so somebody editing the number meets
// the reason not to.
const floorDoc = "docs/coverage-floor.md"

// raiseBy is how far the measured total has to sit above the floor before the
// leg suggests raising it. Small enough that drift is noticed, large enough that
// a one statement change does not produce the suggestion on every run.
const raiseBy = 2.0

// theCaveat is the sentence that travels with the number. It is a constant so
// that the leg cannot print a figure without it.
const theCaveat = "This is the gate tier only. The runner against a real container engine and the " +
	"recipes against real codes are the integration harness, under their own build tag, and " +
	"contribute nothing to this number, so it is a statement about the readers, the model and " +
	"the comparison rather than about the tool. " + floorDoc + " is where that is argued."

// floorRecord is what floorDoc declares.
type floorRecord struct {
	percent float64
	raised  string
}

var (
	floorLine  = regexp.MustCompile(`(?m)^Floor:\s*([0-9]+(?:\.[0-9]+)?)\s*$`)
	raisedLine = regexp.MustCompile(`(?m)^Last raised:\s*(\d{4}-\d{2}-\d{2})\s*$`)
	totalLine  = regexp.MustCompile(`(?m)^total:\s+\(statements\)\s+([0-9]+(?:\.[0-9]+)?)%\s*$`)
)

// parseFloor reads the two fields and refuses anything else. A floor document
// that no longer declares a floor is a failure rather than a floor of zero,
// which is the shape that would let the number fall to nothing while every run
// stayed green.
func parseFloor(doc []byte) (floorRecord, error) {
	m := floorLine.FindSubmatch(doc)
	if m == nil {
		return floorRecord{}, fmt.Errorf("%s declares no `Floor:` line, so there is no floor to compare against", floorDoc)
	}
	percent, err := strconv.ParseFloat(string(m[1]), 64)
	if err != nil {
		return floorRecord{}, fmt.Errorf("%s declares a floor that is not a number: %v", floorDoc, err)
	}
	r := raisedLine.FindSubmatch(doc)
	if r == nil {
		return floorRecord{}, fmt.Errorf("%s declares no `Last raised:` date, so nothing says when the floor was last moved", floorDoc)
	}
	return floorRecord{percent: percent, raised: string(r[1])}, nil
}

// coverageArgs is the test invocation, in one place so that a test can assert
// what it asks for. It names no build tag, which is the whole of how the number
// is kept to the gate tier: a run carrying `-tags integration` would count tests
// that start containers and inflate the figure without covering a line more of
// what the gate proves.
func coverageArgs(profile string) []string {
	return []string{"test", "-coverprofile=" + profile, "./..."}
}

// coverageLeg measures the gate tier and judges it against the floor.
func coverageLeg(root string) outcome {
	if _, err := exec.LookPath("go"); err != nil {
		return skip("no `go` on PATH, so nothing on this run measured coverage. " +
			"Asking for it costs a Go toolchain at the version go.mod declares and a writable temporary directory.")
	}

	doc, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(floorDoc)))
	if err != nil {
		return fail("could not read the floor: " + err.Error())
	}
	floor, err := parseFloor(doc)
	if err != nil {
		return fail(err.Error())
	}

	dir, err := os.MkdirTemp("", "gate-coverage-")
	if err != nil {
		return fail("could not make a directory to write the profile into: " + err.Error())
	}
	defer os.RemoveAll(dir)
	profile := filepath.Join(dir, "coverage.out")

	if o := command(root, "go", coverageArgs(profile)...); o.verdict != passed {
		return o
	}

	funcs, err := coverFunc(root, profile)
	if err != nil {
		return fail(err.Error())
	}
	total, err := totalFrom(funcs)
	if err != nil {
		return fail(err.Error())
	}
	return judgeCoverage(total, floor)
}

// coverFunc turns the profile into the per function listing whose last line is
// the total. The total is taken from the toolchain rather than computed here,
// so the number this leg refuses on is the number `go tool cover` reports.
func coverFunc(root, profile string) (string, error) {
	cmd := exec.Command("go", "tool", "cover", "-func="+profile)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go tool cover -func: %v\n%s", err, strings.TrimRight(string(out), "\r\n"))
	}
	return string(out), nil
}

// totalFrom reads the total out of that listing, and refuses a listing with no
// total rather than defaulting to zero or to a hundred. Both defaults are a
// verdict about a measurement that did not happen.
func totalFrom(funcs string) (float64, error) {
	m := totalLine.FindStringSubmatch(strings.ReplaceAll(funcs, "\r\n", "\n"))
	if m == nil {
		return 0, fmt.Errorf("the coverage listing carries no total line, so nothing was measured to compare against the floor:\n%s",
			strings.TrimRight(funcs, "\n"))
	}
	return strconv.ParseFloat(m[1], 64)
}

// judgeCoverage is the verdict, separated from the running so that every case
// below it can be proven without a toolchain.
func judgeCoverage(total float64, floor floorRecord) outcome {
	if total < floor.percent {
		return fail(fmt.Sprintf(
			"coverage is %.1f%% of statements and the floor is %.1f%%, last raised %s.\n\n"+
				"%s\n\n"+
				"Cover what the change added, or argue the floor down in %s with the reason. "+
				"Editing the number to match the run is how a floor stops being one.",
			total, floor.percent, floor.raised, theCaveat, floorDoc))
	}
	detail := fmt.Sprintf("%.1f%% of statements, against a floor of %.1f%% last raised %s. %s",
		total, floor.percent, floor.raised, theCaveat)
	if total >= floor.percent+raiseBy {
		detail += fmt.Sprintf("\n\nThe measured total is %.1f points above the floor. Raising it is two edits in %s.",
			total-floor.percent, floorDoc)
	}
	return note(detail)
}
