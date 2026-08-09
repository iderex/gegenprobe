package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// The supply chain score is published whether or not anybody reads it, and a
// published number with nothing behind it is read as a verdict somebody
// reached. The register answers every check the tool reports, once, with a
// reason and with the condition that retires an acceptance.
//
// Its failure mode is the tool moving underneath it. A check added upstream
// arrives with no row and the register goes on reading as a complete answer; a
// check retired upstream leaves a row nobody will ever revisit. This leg refuses
// both, in both directions, against the output recorded in the document itself.
//
// What it cannot do is read a live run. That would be a gate leg making a
// network call, which this tree does not have, so the recording is a paste and
// the comparison is between two halves of one file. What that still catches is
// half an edit: a recording brought up to date without the triage being
// revisited. The document says so in its own last section rather than leaving a
// reader to find the bound here.

// registerDoc is the register. It carries the recorded output and the rows, so
// the whole subject of this leg is one file somebody reviews.
const registerDoc = "docs/supply-chain.md"

// outputHeading is the heading the recorded output sits under, and how the block
// is found. Matching on the heading rather than on a position means a section
// added above it does not move the block out from under the leg.
const outputHeading = "## The recorded output"

// registerHeaderCell is the first cell of the table's header line, found the
// same way and for the same reason.
const registerHeaderCell = "Check"

// registerColumns is how many cells a row carries. Named rather than written as
// a number at each use, because a column added to the document without one added
// here is the drift this leg exists against.
const registerColumns = 6

// The three outcomes. A check is answered by exactly one of them: the thing it
// asks for is in the tree, it is a debt with a condition that retires it, or it
// is about something this project does not have.
const (
	outcomeFixed         = "fixed"
	outcomeAccepted      = "accepted"
	outcomeNotApplicable = "not applicable"
)

var (
	takenOutputField  = regexp.MustCompile(`(?m)^Output taken:\s*(\d{4}-\d{2}-\d{2})\s*$`)
	toolVersionField  = regexp.MustCompile(`(?m)^Scorecard version:\s*(\S+)\s*$`)
	scoredCommitField = regexp.MustCompile(`(?m)^Scored commit:\s*([0-9a-f]{40})\s*$`)
	recordedLine      = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9-]*) (-?\d+)$`)
)

// registerRow is one line of the table. It carries the line number so a refusal
// points at the row a reader has to edit rather than at the file.
type registerRow struct {
	line    int
	check   string
	score   string
	outcome string
	retires string
	issue   string
	reason  string
}

// supplyChainLeg judges the register against the output recorded beside it.
func supplyChainLeg(root string) outcome {
	path := filepath.Join(root, filepath.FromSlash(registerDoc))
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return fail(registerDoc + " is not in this tree, so no check the supply chain score reports has an outcome anywhere.")
	}
	if err != nil {
		return fail(err.Error())
	}
	body := strings.ReplaceAll(string(raw), "\r\n", "\n")

	var refused []string
	report := func(line int, why string) {
		refused = append(refused, fmt.Sprintf("%s:%d: %s", registerDoc, line, why))
	}

	taken := ""
	if m := takenOutputField.FindStringSubmatch(body); m != nil {
		taken = m[1]
	} else {
		report(1, "no `Output taken: YYYY-MM-DD` field, so the register makes no claim about when the output below it was recorded")
	}
	version := ""
	if m := toolVersionField.FindStringSubmatch(body); m != nil {
		version = m[1]
	} else {
		report(1, "no `Scorecard version:` field, so nothing says which version of the tool produced the recorded output")
	}
	if scoredCommitField.FindStringSubmatch(body) == nil {
		report(1, "no `Scored commit:` field holding a full commit sha, so nothing says which commit was scored")
	}

	recorded, found := recordedOutput(body, report)
	if !found {
		report(1, "no fenced block under `"+outputHeading+"`, so the register is checked against no output at all")
	}

	rows, table := registerRows(body)
	if !table {
		report(1, "no table whose first column is "+parityQuote(registerHeaderCell))
	}

	seen := map[string]int{}
	for _, r := range rows {
		judgeRegisterRow(r, recorded, seen, report)
	}
	for _, name := range sortedNames(recorded) {
		if _, ok := seen[name]; !ok {
			report(1, "the recorded output names "+parityQuote(name)+" and no row answers it, so a check the score reports has no outcome here")
		}
	}

	if len(refused) > 0 {
		return fail(strings.Join(refused, "\n") +
			"\n\nEvery check the recorded output names carries one outcome, and every row" +
			"\nanswers a check the output names. Re-record the output with the commands " +
			registerDoc + "\ncarries, then answer what moved.")
	}
	if len(recorded) == 0 {
		return skip(registerDoc + " records no check, so nothing was answered")
	}
	fixed, accepted, na := countOutcomes(rows)
	return note(fmt.Sprintf("%d check(s) answered against the output taken %s from Scorecard %s: %d fixed, %d accepted with what retires each, %d not applicable. The output is a recording in the tree and no leg here reads a live score.",
		len(recorded), taken, version, fixed, accepted, na))
}

// judgeRegisterRow holds every rule about one row. The rule that matters most is
// the one about an acceptance: a debt with no condition that retires it is a
// dispensation, and the two read the same on the page.
func judgeRegisterRow(r registerRow, recorded map[string]int, seen map[string]int, report func(int, string)) {
	if r.check == "" {
		report(r.line, "the row names no check")
		return
	}
	if first, ok := seen[r.check]; ok {
		report(r.line, "a second row answers "+parityQuote(r.check)+", which line "+fmt.Sprint(first)+" already answers")
		return
	}
	seen[r.check] = r.line

	score, known := recorded[r.check]
	if !known {
		report(r.line, "row "+parityQuote(r.check)+" answers a check the recorded output does not name")
	} else if r.score != strconv.Itoa(score) {
		report(r.line, "row "+parityQuote(r.check)+" says "+parityQuote(r.score)+" where the recorded output says "+
			parityQuote(strconv.Itoa(score))+", so the output was re-recorded and the triage was not revisited")
	}

	if r.reason == "" || r.reason == "-" {
		report(r.line, "row "+parityQuote(r.check)+" gives no reason, and a blank cell reads on the page exactly like an answer somebody wrote")
	}

	retires := r.retires != "" && r.retires != "-"
	switch r.outcome {
	case outcomeFixed, outcomeNotApplicable:
		if retires {
			report(r.line, "row "+parityQuote(r.check)+" is "+parityQuote(r.outcome)+" and names something that would retire it, which only an acceptance carries")
		}
	case outcomeAccepted:
		if !retires {
			report(r.line, "row "+parityQuote(r.check)+" is accepted and names nothing that would retire the acceptance, which makes it a dispensation rather than a debt")
		}
	default:
		report(r.line, "row "+parityQuote(r.check)+" says "+parityQuote(r.outcome)+" in the outcome column, which is none of "+
			parityQuote(outcomeFixed)+", "+parityQuote(outcomeAccepted)+" or "+parityQuote(outcomeNotApplicable))
	}

	if r.issue != "-" {
		for _, ref := range strings.Split(r.issue, ";") {
			if ref := strings.TrimSpace(ref); !issueRef.MatchString(ref) {
				report(r.line, "row "+parityQuote(r.check)+" names "+parityQuote(ref)+" in the issue column, which is not #NNN")
			}
		}
	}
}

// recordedOutput reads the block the register is judged against. A line inside
// it that is not a check and a score is reported rather than skipped, because a
// block the leg reads half of is a register checked against half an output.
func recordedOutput(body string, report func(int, string)) (map[string]int, bool) {
	lines := strings.Split(body, "\n")
	start := -1
	for i, l := range lines {
		if strings.TrimRight(l, " ") == outputHeading {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, false
	}

	out := map[string]int{}
	inside := false
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "```") {
			if !inside {
				inside = true
				continue
			}
			return out, true
		}
		if !inside || strings.TrimSpace(lines[i]) == "" {
			continue
		}
		m := recordedLine.FindStringSubmatch(strings.TrimRight(lines[i], " "))
		if m == nil {
			report(i+1, "the recorded output holds "+parityQuote(lines[i])+", which is not a check name and a score")
			continue
		}
		score, err := strconv.Atoi(m[2])
		if err != nil {
			report(i+1, "the recorded output holds "+parityQuote(lines[i])+", whose score is not a number")
			continue
		}
		if _, ok := out[m[1]]; ok {
			report(i+1, "the recorded output names "+parityQuote(m[1])+" twice")
			continue
		}
		out[m[1]] = score
	}
	if inside {
		report(start+1, "the fenced block under `"+outputHeading+"` is never closed")
		return out, true
	}
	return nil, false
}

// registerRows finds the table by its header cell and returns the body rows
// under it. A document with no such table returns false rather than no rows,
// because a table nobody can find and a table with nothing in it are different
// faults.
func registerRows(body string) ([]registerRow, bool) {
	lines := strings.Split(body, "\n")
	start := -1
	for i, l := range lines {
		if cells := splitCells(l); len(cells) > 0 && cells[0] == registerHeaderCell {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, false
	}

	var out []registerRow
	for i := start + 2; i < len(lines); i++ {
		cells := splitCells(lines[i])
		if len(cells) == 0 {
			break
		}
		if len(cells) != registerColumns {
			out = append(out, registerRow{line: i + 1, check: fmt.Sprintf("(%d cells rather than %d)", len(cells), registerColumns)})
			continue
		}
		out = append(out, registerRow{
			line: i + 1, check: cells[0], score: cells[1], outcome: cells[2],
			retires: cells[3], issue: cells[4], reason: cells[5],
		})
	}
	return out, true
}

// sortedNames keeps the refusals in one order however the map was filled, so a
// failure a reader has seen once reads the same the next time.
func sortedNames(recorded map[string]int) []string {
	out := make([]string, 0, len(recorded))
	for name := range recorded {
		out = append(out, name)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func countOutcomes(rows []registerRow) (fixed, accepted, na int) {
	for _, r := range rows {
		switch r.outcome {
		case outcomeFixed:
			fixed++
		case outcomeAccepted:
			accepted++
		case outcomeNotApplicable:
			na++
		}
	}
	return fixed, accepted, na
}
