package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// The decision record format is fixed by docs/decisions/0000-decision-records.md.
// That record describes the format; this leg is what refuses a record breaking
// it, which is the difference between a rule and an explanation of one. Until
// this landed, a malformed record reached the default branch exactly as a well
// formed one did, and 0000 says so about itself.
//
// It reads the files as text rather than through a Markdown parser. What is
// judged is a heading, a run of Field: value lines and the order of four level
// two headings, all of which are decidable line by line, and a parser would add
// a dependency this tree does not carry for no property it does not already
// have. The cost is that a section heading written inside a fenced code block
// would be counted as a section, which no record here does.
//
// The generator in tools/decisionindex reads the same files for a different
// purpose, which is to render the index, and it carries its own reader for the
// fields it puts in the table. So parts of the format are read in two places.
// That overlap is real and is not removed here: this leg is the judge of the
// whole format, the generator reads the six fields it renders, and neither is
// derived from the other.

var (
	// recordFile is NNNN-short-slug.md and nothing else. README.md is the index
	// rather than a record, and any other Markdown file in the directory is
	// refused below rather than skipped, because a record whose filename is
	// wrong is a record no route in this tree looks at.
	recordFile = regexp.MustCompile(`^([0-9]{4})-[a-z0-9]+(?:-[a-z0-9]+)*\.md$`)

	// recordHeading is the first line of a record, "# NNNN. Title".
	recordHeading = regexp.MustCompile(`^# ([0-9]{4})\. (.+)$`)

	// sectionHeading is a level two heading. Deeper headings are free, per 0000.
	sectionHeading = regexp.MustCompile(`^## (.+?)\s*$`)

	// recordNumber is what a Number or a Superseded-By field has to look like.
	recordNumber = regexp.MustCompile(`^[0-9]{4}$`)

	// indexLink takes the target out of a Markdown link, so the index can be
	// compared against the directory without rendering it.
	indexLink = regexp.MustCompile(`\]\(([^)]+)\)`)
)

// requiredFields are the four 0000 requires in every record, in the order a
// failure lists them.
var requiredFields = []string{"Number", "Title", "Status", "Date"}

// permittedStatuses is the set 0000 fixes. Anything else is refused rather than
// carried through into an index or a reader's head as if it meant something.
var permittedStatuses = []string{"proposed", "accepted", "superseded"}

// requiredSections are the four level two sections, in the order the format
// fixes. The order is part of the format because the cost section is the one
// everybody skips, and a fixed position makes an absent one visible.
var requiredSections = []string{"What was decided", "Why", "What was rejected", "What this costs"}

// decisionRecordsLeg refuses any record under docs/decisions/ that breaks the
// format, and an index that disagrees with the directory in either direction.
func decisionRecordsLeg(root string) outcome {
	dir := filepath.Join(root, "docs", "decisions")
	problems, examined, err := judgeDecisions(dir)
	switch {
	case os.IsNotExist(err):
		return skip("there is no docs/decisions directory in this tree, so no record was read")
	case err != nil:
		return fail(err.Error())
	case len(problems) > 0:
		return fail(strings.Join(problems, "\n") +
			"\n\nThe format is fixed by docs/decisions/0000-decision-records.md." +
			"\nAfter repairing a record, regenerate the index: go run ./tools/decisionindex")
	case examined == 0:
		return skip("there is no decision record under docs/decisions, so nothing was judged")
	}
	return note(fmt.Sprintf("%d record(s) read, and the index lists exactly those %d.", examined, examined))
}

// judgeDecisions is the leg with the reporting taken off it, so a test can point
// it at a directory of fixtures instead of at a repository.
func judgeDecisions(dir string) (problems []string, examined int, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, err
	}

	var (
		found   []recordFacts
		byNames = map[string]bool{}
		report  = func(file, detail string) {
			problems = append(problems, filepath.ToSlash(filepath.Join(dir, file))+": "+detail)
		}
	)

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		if e.Name() == "README.md" {
			continue
		}
		m := recordFile.FindStringSubmatch(e.Name())
		if m == nil {
			report(e.Name(), "is a Markdown file under docs/decisions whose name is not NNNN-short-slug.md, so no route in this tree reads it as a record")
			continue
		}
		byNames[e.Name()] = true
		facts, ds := judgeRecord(filepath.Join(dir, e.Name()), m[1])
		for _, d := range ds {
			report(e.Name(), d)
		}
		found = append(found, facts)
		examined++
	}

	// A duplicated number is checked across records rather than inside one,
	// because neither of the two files is wrong on its own.
	numbers := map[string][]string{}
	for _, r := range found {
		if r.number != "" {
			numbers[r.number] = append(numbers[r.number], r.file)
		}
	}
	for number, files := range numbers {
		if len(files) > 1 {
			sort.Strings(files)
			problems = append(problems, fmt.Sprintf("%s: number %s is used by %s, and a number that is used twice identifies neither record",
				filepath.ToSlash(dir), number, strings.Join(files, " and ")))
		}
	}

	// A forward pointer at a record that does not exist reads as a supersession
	// that happened, which is the one thing the register must not say wrongly.
	for _, r := range found {
		if r.supersededBy != "" && numbers[r.supersededBy] == nil {
			report(r.file, fmt.Sprintf("is superseded by %s, and no record with that number is in this directory", r.supersededBy))
		}
	}

	problems = append(problems, judgeIndex(dir, byNames)...)
	sort.Strings(problems)
	return problems, examined, nil
}

// recordFacts is what the cross record checks need out of one record: which
// number it claims, and which record it points forward at. They are carried out
// of judgeRecord rather than checked inside it, because neither question can be
// answered until every record in the directory has been read.
type recordFacts struct {
	file         string
	number       string
	supersededBy string
}

// judgeRecord reads one record and returns what the directory level checks need
// from it along with what is wrong with it. The facts come back even where the
// record has problems, so a duplicate number is still reported for a record that
// is also malformed in some other way.
func judgeRecord(path, numberFromName string) (facts recordFacts, problems []string) {
	facts.file = filepath.Base(path)
	raw, err := os.ReadFile(path)
	if err != nil {
		return facts, []string{err.Error()}
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")

	// The heading.
	var headingTitle string
	if h := recordHeading.FindStringSubmatch(lines[0]); h == nil {
		problems = append(problems, `first line is not "# NNNN. Title"`)
	} else {
		headingTitle = h[2]
		if h[1] != numberFromName {
			problems = append(problems, fmt.Sprintf("heading says %s and the filename says %s", h[1], numberFromName))
		}
	}

	fields, fieldProblems := headerBlock(lines)
	problems = append(problems, fieldProblems...)

	for _, f := range requiredFields {
		if fields[f] == "" {
			problems = append(problems, "header field "+f+" is missing")
		}
	}
	number := fields["Number"]
	status := fields["Status"]
	facts.number = number

	if number != "" {
		if !recordNumber.MatchString(number) {
			problems = append(problems, fmt.Sprintf("Number is %q, which is not four digits", number))
		} else if number != numberFromName {
			problems = append(problems, fmt.Sprintf("Number is %s and the filename says %s", number, numberFromName))
		}
	}
	if t := fields["Title"]; t != "" && headingTitle != "" && t != headingTitle {
		problems = append(problems, fmt.Sprintf("heading title %q and Title field %q differ", headingTitle, t))
	}
	if d := fields["Date"]; d != "" {
		if _, err := time.Parse("2006-01-02", d); err != nil {
			problems = append(problems, fmt.Sprintf("Date is %q, which is not a YYYY-MM-DD date", d))
		}
	}
	if status != "" && !contains(permittedStatuses, status) {
		problems = append(problems, fmt.Sprintf("Status is %q, which is not one of %s", status, strings.Join(permittedStatuses, ", ")))
	}

	// Superseded-By is required when and only when the status is superseded,
	// both halves, so a stale pointer on a record that came back into force is
	// refused as well as a missing one.
	pointer := fields["Superseded-By"]
	switch {
	case status == "superseded" && pointer == "":
		problems = append(problems, "status is superseded and Superseded-By names no record, so the register points forward from nothing")
	case status != "superseded" && pointer != "":
		problems = append(problems, fmt.Sprintf("Superseded-By is %s while the status is %q, and the format carries that field only on a superseded record", pointer, status))
	case pointer != "" && !recordNumber.MatchString(pointer):
		problems = append(problems, fmt.Sprintf("Superseded-By is %q, which is not four digits", pointer))
	case pointer != "":
		facts.supersededBy = pointer
	}

	problems = append(problems, judgeSections(lines)...)
	return facts, problems
}

// headerBlock reads the run of "Field: value" lines that follows the heading and
// the blank line under it, stopping at the next blank line, which is where 0000
// says the block ends.
func headerBlock(lines []string) (fields map[string]string, problems []string) {
	fields = map[string]string{}
	i := 1
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	for ; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			break
		}
		key, value, ok := strings.Cut(lines[i], ": ")
		if !ok {
			problems = append(problems, fmt.Sprintf("header line %q is not \"Field: value\"", lines[i]))
			continue
		}
		key = strings.TrimSpace(key)
		if _, seen := fields[key]; seen {
			problems = append(problems, "header field "+key+" appears twice")
		}
		fields[key] = strings.TrimSpace(value)
	}
	return fields, problems
}

// judgeSections requires exactly the four level two headings the format fixes,
// in that order, each with something under it.
func judgeSections(lines []string) (problems []string) {
	var got []string
	content := map[string]bool{}
	current := ""
	for _, line := range lines {
		if m := sectionHeading.FindStringSubmatch(line); m != nil {
			current = m[1]
			got = append(got, current)
			continue
		}
		if current != "" && strings.TrimSpace(line) != "" {
			content[current] = true
		}
	}

	if !equalStrings(got, requiredSections) {
		for _, want := range requiredSections {
			if !contains(got, want) {
				problems = append(problems, fmt.Sprintf("has no %q section", want))
			}
		}
		for _, have := range got {
			if !contains(requiredSections, have) {
				problems = append(problems, fmt.Sprintf("carries a level two heading %q, and the format defines no section by that name", have))
			}
		}
		if len(problems) == 0 {
			problems = append(problems, fmt.Sprintf("has its sections in the order %s, and the format fixes %s",
				strings.Join(quoteAll(got), ", "), strings.Join(quoteAll(requiredSections), ", ")))
		}
		return problems
	}

	for _, want := range requiredSections {
		if !content[want] {
			problems = append(problems, fmt.Sprintf("section %q is empty, and an empty section is a record that has not finished thinking", want))
		}
	}
	return problems
}

// judgeIndex compares docs/decisions/README.md against the records beside it, in
// both directions. The direction that goes stale in silence is the second one:
// a record added without a line in the index is invisible to every reader who
// arrives through the index.
func judgeIndex(dir string, records map[string]bool) (problems []string) {
	path := filepath.Join(dir, "README.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{filepath.ToSlash(path) + ": the index is missing, so nothing lists the records beside it"}
		}
		return []string{err.Error()}
	}

	listed := map[string]bool{}
	for _, m := range indexLink.FindAllStringSubmatch(string(raw), -1) {
		if recordFile.MatchString(m[1]) {
			listed[m[1]] = true
		}
	}

	for name := range listed {
		if !records[name] {
			problems = append(problems, fmt.Sprintf("%s: lists %s, and no such record is in this directory", filepath.ToSlash(path), name))
		}
	}
	for name := range records {
		if !listed[name] {
			problems = append(problems, fmt.Sprintf("%s: does not list %s, so a reader arriving through the index never reaches it", filepath.ToSlash(path), name))
		}
	}
	return problems
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func quoteAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, fmt.Sprintf("%q", s))
	}
	return out
}
