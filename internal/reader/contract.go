package reader

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/iderex/gegenprobe/internal/model"
)

// Requirement is one thing every reader has to do, named so that a failure says
// which of them moved rather than only that something did.
type Requirement string

const (
	// RequirementTruncation is the first thing a reader meets in the field. A
	// run killed halfway leaves a file that stops mid table, and a reader that
	// returns what it managed to parse hands the comparison a table missing
	// its tail with nothing marking the cut.
	RequirementTruncation Requirement = "truncated-file-is-an-error-naming-the-line"

	// RequirementAbsence is 0011 arriving at the one place absence is created.
	// Every state in the model is put there by a reader, so a reader writing
	// zero for a field the code did not print is where a blank enters the
	// bundle, and nothing downstream can tell it from a computed zero.
	RequirementAbsence Requirement = "a-field-the-code-did-not-print-is-a-state"

	// RequirementLabel is 0004's rule that a code's own label is kept verbatim
	// and unparsed. A reader that tidies a label is the earliest point at which
	// this bench could invent agreement between two codes.
	RequirementLabel Requirement = "a-label-reaches-the-model-verbatim"

	// RequirementQuantity is 0004 and 0008 together: a number carries the unit
	// it is in, the count of digits the code actually wrote, and the characters
	// the code printed, or it is a number somebody downstream has to remember
	// something about.
	RequirementQuantity Requirement = "every-number-carries-its-unit-its-significance-and-what-was-printed"

	// RequirementStability is what makes a bundle comparable with itself. A
	// reader whose output depends on anything but its input puts that
	// dependence into every artefact downstream of it, where it reads as
	// physics.
	RequirementStability Requirement = "reading-the-same-file-twice-produces-identical-output"

	// RequirementForeign is the failure that produces plausible nonsense rather
	// than an error. Several of these formats are near enough to each other
	// that the wrong reader parses a file into numbers instead of refusing it.
	RequirementForeign Requirement = "a-file-from-another-code-is-refused-rather-than-parsed"

	// RequirementDecidable is the one that keeps the other six from passing on
	// an absence. Each of them is decided against a recorded file, so a reader
	// registered without one of the four has not met that requirement, it has
	// removed the evidence for it.
	RequirementDecidable Requirement = "every-requirement-is-decided-against-a-recorded-file"
)

// statements is what each requirement says, in the words a failure is read in.
// They are here rather than in a document because a document restating them
// drifts against the code that decides them.
var statements = map[Requirement]string{
	RequirementTruncation: "a truncated file is an error naming the line the file stopped on, and no partial result",
	RequirementAbsence:    "a field the code did not print is one of the four states of 0011 and never zero",
	RequirementLabel:      "a label reaches the model exactly as the code wrote it",
	RequirementQuantity:   "every numeric value carries its unit, the count of digits the code wrote, and the characters it printed",
	RequirementStability:  "reading the same file twice produces identical output",
	RequirementForeign:    "a file the reader was not written for is refused rather than parsed",
	RequirementDecidable:  "every requirement above is decided against a recorded file the reader is registered with",
}

// requirements is the contract, in the order a failure lists them.
var requirements = []Requirement{
	RequirementTruncation,
	RequirementAbsence,
	RequirementLabel,
	RequirementQuantity,
	RequirementStability,
	RequirementForeign,
	RequirementDecidable,
}

// Requirements returns the contract.
func Requirements() []Requirement { return append([]Requirement{}, requirements...) }

// Statement is what the requirement says.
func (r Requirement) Statement() string { return statements[r] }

// Violation is one requirement one reader did not keep, against the recorded
// file it was decided on.
type Violation struct {
	Requirement Requirement
	Case        Case
	Detail      string
}

func (v Violation) String() string {
	return fmt.Sprintf("%s, on the %s file: %s", v.Requirement, v.Case, v.Detail)
}

// Check judges one reader against the whole contract and returns everything it
// did not keep rather than the first thing, so that somebody writing a reader is
// not led through the requirements one failure at a time.
//
// It reads nothing. The recorded bytes are handed to it, keyed by the case each
// one is, because the only thing in this tree that reads a fixture file is
// internal/fixture and a judgement that read its own evidence would be a
// participant in what it judges.
func Check(r Reader, recorded map[Case][]byte) []Violation {
	var out []Violation
	add := func(req Requirement, c Case, format string, args ...any) {
		out = append(out, Violation{Requirement: req, Case: c, Detail: fmt.Sprintf(format, args...)})
	}

	for _, c := range cases {
		if len(recorded[c]) == 0 {
			add(RequirementDecidable, c, "no bytes are registered for this file, so what it decides is decided against nothing")
		}
	}

	var measured map[string]bool
	if bytes := recorded[WellFormed]; len(bytes) > 0 {
		measured = checkWellFormed(r, bytes, add)
	}
	if bytes := recorded[Truncated]; len(bytes) > 0 {
		checkTruncated(r, bytes, add)
	}
	if bytes := recorded[FieldAbsent]; len(bytes) > 0 {
		checkFieldAbsent(r, bytes, measured, add)
	}
	if bytes := recorded[ForeignCode]; len(bytes) > 0 {
		checkForeign(r, bytes, add)
	}
	return out
}

type reporter func(req Requirement, c Case, format string, args ...any)

// checkWellFormed decides the requirements that need a file the reader is
// written for: the labels, the numbers, and reading it twice.
//
// It returns which cells came back measured, keyed by the path they were reached
// at, because that is what the absent field is decided against. The two files
// differ by one field, so a reader that filled it in produces the same set from
// both, and a set is the only form of that comparison that does not need this
// package to know which field either fixture left out.
func checkWellFormed(r Reader, recorded []byte, add reporter) map[string]bool {
	first, err := r.Read(recorded)
	if err != nil {
		add(RequirementDecidable, WellFormed, "reading it failed with %q, so the requirements decided against it were decided against nothing", err)
		return nil
	}
	if first.empty() {
		add(RequirementDecidable, WellFormed, "it yielded no levels and no transitions, so the requirements decided against it were decided against nothing")
		return nil
	}

	text := string(recorded)
	for _, level := range first.Levels {
		if level.Label == "" {
			continue
		}
		if !strings.Contains(text, level.Label) {
			add(RequirementLabel, WellFormed, "level %q carries label %q, which is not in the file; a label that is not in the bytes is one the reader wrote", level.ID, level.Label)
		}
	}

	measured := map[string]bool{}
	for _, c := range cells(first) {
		checkCell(c, text, WellFormed, add)
		if c.quantity.State == model.Measured {
			measured[c.path] = true
		}
	}

	second, err := r.Read(recorded)
	if err != nil {
		add(RequirementStability, WellFormed, "the first read succeeded and the second failed with %q", err)
		return measured
	}
	if !reflect.DeepEqual(first, second) {
		add(RequirementStability, WellFormed, "two reads of the same bytes produced different tables")
	}
	return measured
}

// checkTruncated decides the truncation requirement, in both halves: the error
// says where the file stopped, and nothing came back beside it.
func checkTruncated(r Reader, recorded []byte, add reporter) {
	tables, err := r.Read(recorded)
	if err == nil {
		add(RequirementTruncation, Truncated, "it read without an error, so a file that stops mid table is a table this reader is willing to hand on")
		return
	}
	var incomplete *Incomplete
	switch {
	case !errors.As(err, &incomplete):
		add(RequirementTruncation, Truncated, "the error is %q, which does not name the line the file stopped on; a truncation is reported as *reader.Incomplete", err)
	case incomplete.Line < 1:
		add(RequirementTruncation, Truncated, "the error names line %d, and a line is counted from one", incomplete.Line)
	}
	if !tables.empty() {
		add(RequirementTruncation, Truncated, "it returned %d level(s) and %d transition(s) beside the error, and a partial result beside an error is one a caller reads either way", len(tables.Levels), len(tables.Transitions))
	}
}

// checkFieldAbsent decides that a field the code did not print arrives as a
// state. A reader that refuses the whole file has answered a different question,
// and so has one that fills the cell in.
//
// The decision is the difference against the well formed file rather than the
// presence of an absent cell anywhere in the tables. Most cells of most readers
// are absent for other reasons: a format that prints no mixing weight leaves
// that cell absent in every file it writes, so a reader could turn the one field
// this fixture removed into a zero and still return absences by the dozen.
func checkFieldAbsent(r Reader, recorded []byte, measured map[string]bool, add reporter) {
	tables, err := r.Read(recorded)
	if err != nil {
		add(RequirementAbsence, FieldAbsent, "reading it failed with %q, and a field the code did not print is a state on one cell rather than a refusal to read the file", err)
		return
	}
	found := cells(tables)
	if len(found) == 0 {
		add(RequirementDecidable, FieldAbsent, "it yielded no cell at all, so nothing here says what happened to the absent field")
		return
	}
	lost := 0
	for _, c := range found {
		checkCell(c, string(recorded), FieldAbsent, add)
		if measured[c.path] && c.quantity.State != model.Measured {
			lost++
		}
	}
	if measured == nil {
		return
	}
	if lost == 0 {
		add(RequirementAbsence, FieldAbsent, "all %d cell(s) that are %q in the well formed file are %q here as well, and the two files differ by a field this one does not carry, so the field the code did not print arrived as a value", len(measured), model.Measured, model.Measured)
	}
}

// checkForeign decides that the wrong file is refused rather than parsed, which
// is the requirement whose failure looks most like a result.
func checkForeign(r Reader, recorded []byte, add reporter) {
	tables, err := r.Read(recorded)
	if err == nil {
		add(RequirementForeign, ForeignCode, "it read without an error, so another code's file parsed into %d level(s) and %d transition(s)", len(tables.Levels), len(tables.Transitions))
		return
	}
	var foreign *Foreign
	if !errors.As(err, &foreign) {
		add(RequirementForeign, ForeignCode, "the error is %q, which does not say the file was written by another code; that refusal is reported as *reader.Foreign", err)
	}
	if !tables.empty() {
		add(RequirementForeign, ForeignCode, "it returned %d level(s) and %d transition(s) beside the error", len(tables.Levels), len(tables.Transitions))
	}
}

// checkCell judges one numeric cell. The model refuses the shapes it can refuse
// on its own, and this adds the one it cannot: that what the cell says the code
// printed is in the file the code printed it in.
func checkCell(c cell, text string, in Case, add reporter) {
	if err := c.quantity.Check(); err != nil {
		add(RequirementQuantity, in, "%s: %s", c.path, err)
		return
	}
	if c.quantity.State != model.Measured {
		return
	}
	if !strings.Contains(text, c.quantity.Printed) {
		add(RequirementQuantity, in, "%s says the code printed %q, which is not in the file; a printed value that is not in the bytes is one the reader rendered", c.path, c.quantity.Printed)
	}
}

// cell is one numeric cell of the model with the path it was reached by, so a
// failure names the field rather than the ordinal of a value in a list.
type cell struct {
	path     string
	quantity model.Quantity
}

// cells walks the tables for every [model.Quantity] in them, by reflection
// rather than by a list of fields. A list here would be a second copy of the
// model, and it would go stale on the first field somebody adds to a level.
func cells(t Tables) []cell {
	var out []cell
	walk(reflect.ValueOf(t), "", &out)
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	return out
}

var quantityType = reflect.TypeOf(model.Quantity{})

func walk(v reflect.Value, path string, out *[]cell) {
	if v.Type() == quantityType {
		*out = append(*out, cell{path: path, quantity: v.Interface().(model.Quantity)})
		return
	}
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			walk(v.Field(i), join(path, field.Name), out)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walk(v.Index(i), fmt.Sprintf("%s[%d]", path, i), out)
		}
	case reflect.Pointer:
		if !v.IsNil() {
			walk(v.Elem(), path, out)
		}
	}
}

func join(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}
