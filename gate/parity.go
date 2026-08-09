package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// The parity table maps every check on the target board onto this one. Its
// failure mode is not being wrong on the day it is written; it is going stale
// afterwards, one renamed leg and one deleted workflow at a time, while every
// row still reads as a statement somebody checked. This leg is what refuses
// that.
//
// It reads in one direction only, and the direction it does not read is named
// in the document rather than left to be discovered. A check this board
// publishes that no row mentions is not refused here, because the published
// name of a job carrying a matrix is not derivable from the workflow file
// without reading the matrix, and two of the names this board publishes belong
// to no job at all.

// parityDoc is the table. It carries both the rows and the two fields the leg
// reads, so the whole subject of this leg is one file somebody reviews.
const parityDoc = "docs/quality-parity.md"

// headerCell is the first cell of the table's header line, and how the table is
// found. Matching on the text rather than on the position means a section added
// above the table does not move it out from under the leg.
const headerCell = "Check on the target board"

// columns is how many cells a row carries. Named rather than written as a
// number at each use, because a column added to the document without one added
// here is the drift this leg exists against.
const columns = 6

var (
	takenField    = regexp.MustCompile(`(?m)^Target list taken:\s*(\d{4}-\d{2}-\d{2})\s*$`)
	requiredField = regexp.MustCompile(`(?m)^Required status checks:\s*(\S.*?)\s*$`)
	issueRef      = regexp.MustCompile(`^#\d+$`)
)

// parityRow is one line of the table. It carries the line number so a refusal
// points at the row a reader has to edit rather than at the file.
type parityRow struct {
	line    int
	check   string
	covers  string
	by      string
	merge   string
	issue   string
	comment string
}

// parityLeg judges the table against this tree. The leg names are taken from
// the run list itself, so a leg renamed in legs.go and not in the document is
// refused by the next run rather than by a reader.
func parityLeg(root string) outcome {
	known := map[string]bool{}
	for _, l := range legs() {
		known[l.name] = true
	}
	return judgeParity(root, known)
}

// judgeParity takes the leg names as an argument so a fixture can be judged
// against a stated set rather than against whatever this repository's run list
// happens to hold on the day the test runs.
func judgeParity(root string, known map[string]bool) outcome {
	path := filepath.Join(root, filepath.FromSlash(parityDoc))
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return fail(parityDoc + " is not in this tree, so no check on the target board is mapped onto this one.")
	}
	if err != nil {
		return fail(err.Error())
	}
	body := strings.ReplaceAll(string(raw), "\r\n", "\n")

	var refused []string
	report := func(line int, why string) {
		refused = append(refused, fmt.Sprintf("%s:%d: %s", parityDoc, line, why))
	}

	taken := ""
	if m := takenField.FindStringSubmatch(body); m != nil {
		taken = m[1]
	} else {
		report(1, "no `Target list taken: YYYY-MM-DD` field, so the table makes no claim about when it was derived")
	}
	required, requiredLine := requiredChecks(body)
	if required == nil {
		report(1, "no `Required status checks:` field, so the merge column is checked against nothing")
	}

	rows, found := tableRows(body)
	if !found {
		report(1, "no table whose first column is "+headerCell)
	}

	for _, r := range rows {
		judgeRow(r, known, required, requiredLine, root, report)
	}

	mentioned, err := unmentionedWorkflows(root, body)
	if err != nil {
		return fail(err.Error())
	}
	for _, w := range mentioned {
		report(1, w+" is a workflow in this tree that the table names nowhere, so a check was added here without a row saying what on the target board it answers")
	}

	if len(refused) > 0 {
		return fail(strings.Join(refused, "\n") +
			"\n\nEvery row names either something here that covers the check or the reason nothing" +
			"\ndoes. Re-derive the target list with the commands " + parityDoc + " carries.")
	}
	if len(rows) == 0 {
		return skip(parityDoc + " holds no row, so no check on the target board was mapped")
	}
	return note(fmt.Sprintf("%d row(s) read against the target list taken %s, %d of them covered here and %d carrying the reason nothing is. No row claims to block a merge, which is what the Required status checks field declares.",
		len(rows), taken, coveredCount(rows), len(rows)-coveredCount(rows)))
}

// judgeRow holds every rule about one row. The rule that matters most is the
// last cheap one: a row with neither a covering entry nor a reason is refused,
// because a blank cell reads on the page exactly like an absence somebody
// considered.
func judgeRow(r parityRow, known map[string]bool, required []string, requiredLine int, root string, report func(int, string)) {
	if r.check == "" {
		report(r.line, "the row names no check on the target board")
	}
	if r.covers == "" || r.covers == "-" {
		report(r.line, "row "+parityQuote(r.check)+" does not say what the check covers there")
	}

	covered := r.by != "-"
	explained := r.comment != "-" && r.comment != ""

	if !covered && !explained {
		report(r.line, "row "+parityQuote(r.check)+" names nothing here that covers it and gives no reason for the deviation")
	}
	if covered {
		for _, entry := range strings.Split(r.by, ";") {
			if why := unresolved(strings.TrimSpace(entry), known, root); why != "" {
				report(r.line, "row "+parityQuote(r.check)+": "+why)
			}
		}
	}

	switch r.merge {
	case "no":
	case "yes":
		if !requiredNames(required, r.check) {
			report(r.line, "row "+parityQuote(r.check)+" says it blocks a merge here, and the Required status checks field on line "+
				fmt.Sprint(requiredLine)+" does not name it")
		}
	default:
		report(r.line, "row "+parityQuote(r.check)+" says "+parityQuote(r.merge)+" in the merge column, which is neither yes nor no")
	}

	if r.issue != "-" {
		for _, ref := range strings.Split(r.issue, ";") {
			if ref := strings.TrimSpace(ref); !issueRef.MatchString(ref) {
				report(r.line, "row "+parityQuote(r.check)+" names "+parityQuote(ref)+" in the issue column, which is not #NNN")
			}
		}
	}
}

// unresolved says why a covering entry names nothing in this tree, or returns
// the empty string where it resolves. The two forms are the two kinds of thing
// that can cover a check here: a leg of this command, and a workflow file.
func unresolved(entry string, known map[string]bool, root string) string {
	switch {
	case strings.HasPrefix(entry, "leg "):
		name := strings.TrimSpace(strings.TrimPrefix(entry, "leg "))
		if !known[name] {
			return parityQuote(name) + " is named as a leg of this command and there is no such leg"
		}
	case strings.HasPrefix(entry, "workflow "):
		rel := strings.TrimSpace(strings.TrimPrefix(entry, "workflow "))
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			return parityQuote(rel) + " is named as a workflow in this tree and there is no such file"
		}
	default:
		return parityQuote(entry) + " is neither `leg <name>` nor `workflow <path>`"
	}
	return ""
}

// tableRows finds the table by its header cell and returns the body rows under
// it. A document with no such table returns false rather than no rows, because
// a table nobody can find and a table with nothing in it are different faults.
func tableRows(body string) ([]parityRow, bool) {
	lines := strings.Split(body, "\n")
	start := -1
	for i, l := range lines {
		if cells := splitCells(l); len(cells) > 0 && cells[0] == headerCell {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, false
	}

	var out []parityRow
	for i := start + 2; i < len(lines); i++ {
		cells := splitCells(lines[i])
		if len(cells) == 0 {
			break
		}
		if len(cells) != columns {
			out = append(out, parityRow{line: i + 1, check: fmt.Sprintf("(%d cells rather than %d)", len(cells), columns)})
			continue
		}
		out = append(out, parityRow{
			line: i + 1, check: cells[0], covers: cells[1], by: cells[2],
			merge: cells[3], issue: cells[4], comment: cells[5],
		})
	}
	return out, true
}

// split turns a table line into its cells, and returns nothing for a line that
// is not one. The separator row under a header is a row of dashes and is not a
// row of the table.
func splitCells(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil
	}
	parts := strings.Split(strings.Trim(line, "|"), "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

// requiredChecks reads the field declaring which checks can stop a merge here,
// and the line it is on so a refusal can point at it. The word none is the
// empty set stated rather than left blank, since a blank field and a field
// nobody wrote are the same bytes.
func requiredChecks(body string) ([]string, int) {
	m := requiredField.FindStringSubmatch(body)
	if m == nil {
		return nil, 0
	}
	line := 1 + strings.Count(body[:strings.Index(body, m[0])], "\n")
	if m[1] == "none" {
		return []string{}, line
	}
	var out []string
	for _, c := range strings.Split(m[1], ";") {
		out = append(out, strings.TrimSpace(c))
	}
	return out, line
}

func requiredNames(required []string, check string) bool {
	for _, c := range required {
		if c == check {
			return true
		}
	}
	return false
}

// unmentionedWorkflows is the half of the reverse direction this leg can decide
// without reading a matrix: a workflow file exists here and no row names it. A
// check added to this board without a row is the drift that turns the table
// into a description of an older tree.
func unmentionedWorkflows(root, body string) ([]string, error) {
	dir := filepath.Join(root, ".github", "workflows")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var missing []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if ext := filepath.Ext(e.Name()); ext != ".yml" && ext != ".yaml" {
			continue
		}
		rel := ".github/workflows/" + e.Name()
		if !strings.Contains(body, rel) {
			missing = append(missing, rel)
		}
	}
	return missing, nil
}

func coveredCount(rows []parityRow) int {
	n := 0
	for _, r := range rows {
		if r.by != "-" && r.by != "" {
			n++
		}
	}
	return n
}

func parityQuote(s string) string { return "`" + s + "`" }
