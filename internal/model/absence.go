package model

import "fmt"

// State is which of the four states of 0011 a cell is in. There is no fifth
// state and no encoding of absence outside this vocabulary: not null, not the
// empty string, not a blank and not zero. Zero is a physically meaningful energy
// and a physically meaningful oscillator strength, so rendering absence as zero
// is a wrong number rather than a shortcut.
type State string

const (
	// Measured is the code produced this value and it is here.
	Measured State = "measured"
	// Declined is the code was asked and did not produce it. A statement about
	// a code.
	Declined State = "declined"
	// NotRequested is the case did not ask for this. A statement about the
	// question, and about no code.
	NotRequested State = "not-requested"
	// Refused is this harness declined to produce or to compare. A statement
	// about this harness.
	Refused State = "refused"
)

// Reason is why a cell is absent. Each absent state has its own vocabulary and
// they do not overlap, which is what stops a reason from quietly moving a cell
// between two states that mean opposite things.
type Reason string

const (
	NotComputed  Reason = "not-computed"
	NotInOutput  Reason = "not-in-output"
	NotConverged Reason = "not-converged"
	CodeFailed   Reason = "code-failed"

	QuantityNotRequested Reason = "quantity-not-requested"
	LevelNotSelected     Reason = "level-not-selected"
	ParticipantNotInCase Reason = "participant-not-in-case"

	Unmatched             Reason = "unmatched"
	Ambiguous             Reason = "ambiguous"
	PhysicsDiffers        Reason = "physics-differs"
	PrecisionInsufficient Reason = "precision-insufficient"
	UnitNotConvertible    Reason = "unit-not-convertible"
	RunIncomplete         Reason = "run-incomplete"
)

// reasons is the vocabulary per state, and it is the whole of it. A reason that
// is not here is refused rather than accepted as a free text note, because 0011
// says an implementation meeting an unlisted case adds the reason and supersedes
// the record instead of picking the nearest one.
var reasons = map[State][]Reason{
	Measured:     nil,
	Declined:     {NotComputed, NotInOutput, NotConverged, CodeFailed},
	NotRequested: {QuantityNotRequested, LevelNotSelected, ParticipantNotInCase},
	Refused:      {Unmatched, Ambiguous, PhysicsDiffers, PrecisionInsufficient, UnitNotConvertible, RunIncomplete},
}

// states is the order the four are written in wherever they are listed, which is
// the order 0011 puts them in and the order aggregation takes the strongest
// absence in.
var states = []State{Measured, Declined, NotRequested, Refused}

// checkAbsence judges a state and reason together, because neither is judgeable
// alone: a reason belongs to exactly one state, and the pair is what a cell
// carries.
//
// The failure it refuses is the one 0011 names as the worst, a cell reaching the
// artefact with no state at all. Such a cell renders as a blank, and a blank
// that could mean any of four things is unreadable rather than weakly readable:
// a reader resolves it in whichever direction they already believed.
func checkAbsence(state State, reason Reason) error {
	known, ok := reasons[state]
	if !ok {
		if state == "" {
			return fmt.Errorf("no state: every cell is in one of the four states of 0011, %s", list(stateNames()))
		}
		return fmt.Errorf("state %q is none of the four states of 0011, %s", state, list(stateNames()))
	}
	if state == Measured {
		if reason != "" {
			return fmt.Errorf("state %q carries reason %q, and a value that is here needs no reason for not being here", state, reason)
		}
		return nil
	}
	if reason == "" {
		return fmt.Errorf("state %q carries no reason, and an absence with no reason is the blank 0011 refuses under another name; one of %s", state, list(reasonNames(state)))
	}
	for _, r := range known {
		if r == reason {
			return nil
		}
	}
	return fmt.Errorf("state %q carries reason %q, which belongs to no state; the reasons for %q are %s", state, reason, state, list(reasonNames(state)))
}

func stateNames() []string {
	out := make([]string, 0, len(states))
	for _, s := range states {
		out = append(out, string(s))
	}
	return out
}

func reasonNames(state State) []string {
	out := make([]string, 0, len(reasons[state]))
	for _, r := range reasons[state] {
		out = append(out, string(r))
	}
	return out
}

// list writes a vocabulary the way a refusal has to: every admissible value
// spelt out, because a reader meeting the refusal is choosing between them and a
// message naming none of them sends them to the source.
func list(values []string) string {
	out := ""
	for i, v := range values {
		switch {
		case i == 0:
		case i == len(values)-1:
			out += " or "
		default:
			out += ", "
		}
		out += `"` + v + `"`
	}
	return out
}
