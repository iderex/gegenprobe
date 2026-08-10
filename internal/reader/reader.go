// Package reader holds the contract every reader satisfies, the registry of the
// readers this build carries, and the judgement that decides whether one of them
// keeps the contract.
//
// A reader turns the bytes one code wrote into the types of internal/model. The
// codes in this bench write in formats that have nothing to do with each other,
// so the readers cannot share an implementation, and the thing they have to
// share instead is how they behave at their edges. Without that, a truncated
// file means five different things depending on which code produced it, and the
// comparison downstream inherits all five.
//
// What the contract requires is written once, in [Requirements], and judged once,
// in [Check]. Each requirement carries the recorded file it is decided against,
// so a reader arrives with four fixtures or it arrives failing.
//
// The judging takes bytes it is handed rather than reading anything itself, the
// same way internal/boundary judges an import graph handed to it. A reader's
// fixtures are loaded by the suite, through internal/fixture, which is the only
// thing in this tree that reads a recorded file.
//
// A reader joins the suite by adding one entry to the list in [Registrations],
// and nothing else changes. That is what makes an unsatisfied contract a failure
// rather than an omission: the suite reaches every entry in that list, so a
// reader cannot be added and left unjudged without deleting the line that added
// it.
//
// No reader is registered in this build. The readers are issues #45 to #49 and
// none of them is in the tree, so the suite in this package covers zero readers
// and says so rather than passing quietly. What is proven today is the
// judgement itself, against doubles that break one requirement each.
package reader

import (
	"fmt"

	"github.com/iderex/gegenprobe/internal/model"
)

// Reader turns one file a code wrote into the model.
//
// It is one method rather than one per kind of file because which kinds a code
// writes is that code's business: FAC writes its levels and its transitions
// into separate files, the last program of the Cowan chain writes several kinds
// of block into one, and a reader that had to declare the set in advance would
// be declaring something about a format rather than reading it.
type Reader interface {
	// Participant is the participant identifier this reader produces tables
	// for, in the spelling a case file uses.
	Participant() string

	// Read turns the recorded bytes of one file into levels and transitions.
	//
	// It returns no tables at all with an error. A partial table beside an
	// error is the shape that gets used: a caller that checks the error is
	// indistinguishable from one that does not, until the day somebody writes
	// the second kind.
	Read(recorded []byte) (Tables, error)
}

// Tables is what one file yielded. Both halves are permitted to be empty,
// because a code writing its levels and its transitions into separate files
// yields one of them per file.
type Tables struct {
	Levels      []model.Level
	Transitions []model.Transition
}

// empty says whether anything was returned at all, which is what the truncation
// and foreign-file requirements assert alongside the error.
func (t Tables) empty() bool { return len(t.Levels) == 0 && len(t.Transitions) == 0 }

// Incomplete is what a reader returns for a file that stops before its table
// does, and it names the line the file stopped on.
//
// The line is a field rather than a sentence because the contract has to decide
// whether it is there. An operator whose run was killed halfway through wants to
// know where the output ends, and a message saying only that the file is
// truncated sends them to count the lines themselves.
type Incomplete struct {
	// Line is the line the file stopped on, counting from one.
	Line int
	// Detail says what was being read when it stopped.
	Detail string
}

func (e *Incomplete) Error() string {
	return fmt.Sprintf("the file stops at line %d: %s", e.Line, e.Detail)
}

// Foreign is what a reader returns for a file another code wrote.
//
// It is a separate type from [Incomplete] because the two are different answers
// to the operator: one file is the right file cut short, the other is the wrong
// file. Collapsing them into one error would leave somebody who pointed the
// harness at the wrong directory reading about truncation.
type Foreign struct {
	// Detail says what the reader expected and what it found instead.
	Detail string
}

func (e *Foreign) Error() string {
	return "this file was not written by this code: " + e.Detail
}

// Case is which of the four recorded files a requirement is decided against.
type Case string

const (
	// WellFormed is a file the reader is written for, complete and with every
	// field the code prints present. It is what the requirements about labels,
	// units, significance and stability are decided against, and it is the one
	// fixture that is not about a failure.
	WellFormed Case = "well-formed"
	// Truncated is the same file stopping mid table.
	Truncated Case = "truncated"
	// FieldAbsent is the same file with a field the code did not print.
	FieldAbsent Case = "field-absent"
	// ForeignCode is a file another code in this bench wrote.
	ForeignCode Case = "foreign-code"
)

// cases is the set a registration carries, and all four are required. The three
// failures are the ones the issue names; the fourth is here because a reader
// judged only against files it has to refuse has never been asked to read one.
var cases = []Case{WellFormed, Truncated, FieldAbsent, ForeignCode}

// Cases returns the recorded files every reader is registered with.
func Cases() []Case { return append([]Case{}, cases...) }

// Registration is one reader and the recorded files it is judged against.
type Registration struct {
	// Reader is the reader itself.
	Reader Reader
	// Fixtures names the file for each case, as a name under this package's
	// own testdata directory. Every case in [Cases] is required, and a
	// registration missing one fails the contract rather than covering less of
	// it.
	Fixtures map[Case]string
}

// Registrations is every reader this build carries.
//
// A reader is a file in this package that adds one entry here. The suite reads
// this list and nothing else, so the entry is the whole of joining it, and a
// reader that is in the tree and not in this list is one the suite does not
// judge. That is the only way to be outside the contract, and it is one line
// long and visible in a diff.
func Registrations() []Registration {
	// The readers are #45 to #49 and none of them is in the tree yet.
	return []Registration{}
}
