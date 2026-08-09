package model

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Unit is the unit a number is in. It is an enumerated value rather than a
// string somebody writes, because the whole point of 0004's canonical unit is
// that a reader can hold two numbers side by side without converting anything,
// and a free text unit puts that back on them.
type Unit string

const (
	// ReciprocalCentimetre is the canonical unit for every energy, which 0004
	// chose because it is what the observational literature and the level
	// databases use.
	ReciprocalCentimetre Unit = "cm^-1"
	// Nanometre is the canonical unit for a wavelength, in vacuum.
	Nanometre Unit = "nm"
	// PerSecond is the canonical unit for a transition probability.
	PerSecond Unit = "s^-1"
	// AtomicUnitsOfLineStrength is e^2 a_0^2, the canonical unit for a line
	// strength, and 0004 carries it only where a code printed one.
	AtomicUnitsOfLineStrength Unit = "e^2a_0^2"
	// Dimensionless is a weighted oscillator strength or a mixing weight,
	// which have no unit and still carry one, so that no field in the model is
	// the one a reader has to remember about.
	Dimensionless Unit = "dimensionless"

	// The rest are units codes print in. They appear in a cell's printed unit
	// and never in its canonical one.
	Electronvolt     Unit = "eV"
	Hartree          Unit = "hartree"
	Rydberg          Unit = "rydberg"
	Angstrom         Unit = "angstrom"
	AtomicUnitOfTime Unit = "atomic-unit-of-time"
)

// canonical is the unit each quantity is stored in after conversion. A unit
// outside this set can be printed by a code and cannot be the canonical unit of
// a cell.
var canonical = []Unit{ReciprocalCentimetre, Nanometre, PerSecond, AtomicUnitsOfLineStrength, Dimensionless}

// units is every unit the model knows, canonical and printed together.
var units = append(append([]Unit{}, canonical...), Electronvolt, Hartree, Rydberg, Angstrom, AtomicUnitOfTime)

// Marker says how the reader arrived at the significance count, which 0008
// requires beside the count itself. A count nobody can trace is a count nobody
// can argue with.
type Marker string

const (
	// Stated is the code documented its own output precision.
	Stated Marker = "stated"
	// Counted is the reader counted the digits in the field.
	Counted Marker = "counted"
	// FixedFormat is the reader took it from the code's fixed format
	// specification.
	FixedFormat Marker = "format"
)

var markers = []Marker{Stated, Counted, FixedFormat}

// Quantity is one numeric cell of the model, and it is the only shape a number
// with physical meaning takes here. It carries the state of 0011, the value the
// code printed with the unit the code used, the value converted to the canonical
// unit of 0004, and the significance count and marker of 0008.
//
// The four are together in one type because splitting them is how each of them
// gets lost. A value without its printed form cannot be checked against the file
// it came from; a value without its unit is a number in whichever unit the
// reader assumed; a value without its significance is printed to more digits
// than the code produced, which is the failure 0008 exists against; and a value
// without a state renders absence as a blank.
//
// Conversion to the canonical unit is not done here. The constants and the
// digits that survive a conversion are #50, and this type is the shape the
// result of that work is carried in.
type Quantity struct {
	State  State  `json:"state"`
	Reason Reason `json:"reason,omitempty"`

	// Printed is the value exactly as the code wrote it, characters and all,
	// so that the question of whether the code said this or whether the
	// harness computed it is answerable from the artefact alone.
	Printed string `json:"printed,omitempty"`
	// PrintedUnit is the unit the code used, which is often not the canonical
	// one.
	PrintedUnit Unit `json:"printed-unit,omitempty"`

	// Value is the value converted to the canonical unit.
	Value float64 `json:"value,omitempty"`
	// Unit is the canonical unit of 0004 for this quantity.
	Unit Unit `json:"unit,omitempty"`

	// Significance is how many significant digits the code actually wrote.
	Significance int `json:"significance,omitempty"`
	// Marker is how the reader arrived at that count.
	Marker Marker `json:"significance-marker,omitempty"`
	// TrailingZeroAmbiguous is set where the code printed an integer with
	// trailing zeros and no decimal point, so the count above is an upper
	// bound. 0008 refuses a guess here and requires the mark to travel with
	// every comparison the value enters.
	TrailingZeroAmbiguous bool `json:"trailing-zero-ambiguous,omitempty"`
}

// Absent is the only way to build a cell that holds no value. It takes the state
// and the reason together because neither is meaningful alone, and it refuses a
// pair 0011 does not permit.
func Absent(state State, reason Reason) (Quantity, error) {
	if state == Measured {
		return Quantity{}, fmt.Errorf("absent cell: %q is the state of a value that is here", Measured)
	}
	if err := checkAbsence(state, reason); err != nil {
		return Quantity{}, fmt.Errorf("absent cell: %w", err)
	}
	return Quantity{State: state, Reason: reason}, nil
}

// Check judges one cell. It is what MarshalJSON calls, and it is exported
// because a reader that builds a cell wants the refusal at the point it built
// the wrong one rather than at the point the bundle is written.
func (q Quantity) Check() error {
	if err := checkAbsence(q.State, q.Reason); err != nil {
		return err
	}
	if q.State != Measured {
		if q.Printed != "" || q.Value != 0 || q.Unit != "" || q.Significance != 0 {
			return fmt.Errorf("state %q carries a value, and an absent cell holds none: no route in this model turns an absence into a measurement", q.State)
		}
		return nil
	}
	if q.Printed == "" {
		return fmt.Errorf("state %q carries no printed value, so nothing says what the code wrote", Measured)
	}
	if !known(units, q.PrintedUnit) {
		return fmt.Errorf("printed unit %q is not a unit this model knows", q.PrintedUnit)
	}
	if !known(canonical, q.Unit) {
		return fmt.Errorf("unit %q is not a canonical unit of 0004; a cell is stored in one of %s", q.Unit, list(unitNames(canonical)))
	}
	if q.Significance < 1 {
		return fmt.Errorf("significance %d: a measured value carries the count of digits the code actually wrote, which 0008 requires and which is at least one", q.Significance)
	}
	if !known(markers, q.Marker) {
		return fmt.Errorf("significance marker %q says nothing about how the count was arrived at; 0008 admits %s", q.Marker, list(markerNames()))
	}
	return nil
}

// MarshalJSON refuses a cell the model does not permit, so that the refusal is
// the type's rather than a reviewer's. Every route that writes a bundle goes
// through here, and there is no second encoding of a cell for one of them to
// use instead.
func (q Quantity) MarshalJSON() ([]byte, error) {
	if err := q.Check(); err != nil {
		return nil, fmt.Errorf("cell: %w", err)
	}
	return json.Marshal(cell(q))
}

// UnmarshalJSON refuses the same shapes on the way in. A bundle written by
// something other than this code is read under the same rules as one written by
// it, which is what stops a hand edited artefact from entering the comparison
// with a cell nothing judged.
func (q *Quantity) UnmarshalJSON(b []byte) error {
	var c cell
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		return fmt.Errorf("cell: %w", err)
	}
	out := Quantity(c)
	if err := out.Check(); err != nil {
		return fmt.Errorf("cell: %w", err)
	}
	*q = out
	return nil
}

// cell is Quantity without its methods, which is what breaks the recursion the
// encoder would otherwise fall into.
type cell Quantity

func known[T comparable](in []T, want T) bool {
	for _, v := range in {
		if v == want {
			return true
		}
	}
	return false
}

func unitNames(in []Unit) []string {
	out := make([]string, 0, len(in))
	for _, u := range in {
		out = append(out, string(u))
	}
	return out
}

func markerNames() []string {
	out := make([]string, 0, len(markers))
	for _, m := range markers {
		out = append(out, string(m))
	}
	return out
}
