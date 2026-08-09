package model

import (
	"encoding/json"
	"strings"
	"testing"
)

// measured is the cell every case below is one field away from. It is a level
// energy a code printed in electronvolts, converted to the canonical unit, with
// the digit count the reader took from the field.
func measured() Quantity {
	return Quantity{
		State:        Measured,
		Printed:      "1.23450E+03",
		PrintedUnit:  Electronvolt,
		Value:        9956874.2,
		Unit:         ReciprocalCentimetre,
		Significance: 6,
		Marker:       Counted,
	}
}

func TestAMeasuredCellCarriesWhatTheCodeWroteAndWhatItConvertsTo(t *testing.T) {
	b, err := json.Marshal(measured())
	if err != nil {
		t.Fatalf("a complete cell was refused: %v", err)
	}
	for _, want := range []string{
		`"state":"measured"`,
		`"printed":"1.23450E+03"`,
		`"printed-unit":"eV"`,
		`"unit":"cm^-1"`,
		`"significance":6`,
		`"significance-marker":"counted"`,
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("the encoded cell does not carry %s:\n%s", want, b)
		}
	}
}

// The refusal the issue asks for by name: a field cannot be marked absent
// without a state from 0011, and the type refuses it rather than a reviewer.
func TestACellWithNoStateIsRefused(t *testing.T) {
	q := measured()
	q.State = ""

	_, err := json.Marshal(q)

	if err == nil {
		t.Fatal("a cell with no state encoded, so absence could reach an artefact as a blank")
	}
	for _, want := range []string{"no state", `"measured"`, `"declined"`, `"not-requested"`, `"refused"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not carry %q: %v", want, err)
		}
	}
}

// Every other way a state and a reason can fail to say what a cell is. Each is
// one field away from a cell that encodes.
func TestACellWhoseStateAndReasonDisagreeIsRefused(t *testing.T) {
	for _, c := range []struct {
		name string
		q    Quantity
		want string
	}{
		{
			"an absence with no reason",
			Quantity{State: Refused},
			"carries no reason",
		},
		{
			"a reason from another state's vocabulary",
			Quantity{State: Refused, Reason: NotConverged},
			"belongs to no state",
		},
		{
			"a state outside the four",
			Quantity{State: "missing", Reason: Unmatched},
			"is none of the four states",
		},
		{
			"a measured cell carrying a reason",
			func() Quantity { q := measured(); q.Reason = NotConverged; return q }(),
			"needs no reason for not being here",
		},
		{
			"an absent cell carrying a value",
			Quantity{State: Declined, Reason: NotConverged, Printed: "1.5", Value: 1.5, Unit: ReciprocalCentimetre, Significance: 2, Marker: Counted},
			"no route in this model turns an absence into a measurement",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := json.Marshal(c.q); err == nil {
				t.Fatalf("%s encoded", c.name)
			} else if !strings.Contains(err.Error(), c.want) {
				t.Errorf("the refusal does not carry %q: %v", c.want, err)
			}
		})
	}
}

// A measured cell that has lost its unit or its digit count is the failure 0008
// exists against, and it is refused at the same place.
func TestAMeasuredCellMissingItsUnitOrItsSignificanceIsRefused(t *testing.T) {
	for _, c := range []struct {
		name   string
		change func(*Quantity)
		want   string
	}{
		{"no printed value", func(q *Quantity) { q.Printed = "" }, "nothing says what the code wrote"},
		{"no printed unit", func(q *Quantity) { q.PrintedUnit = "" }, "is not a unit this model knows"},
		{"a canonical unit that is not one", func(q *Quantity) { q.Unit = Electronvolt }, "is not a canonical unit of 0004"},
		{"no significance count", func(q *Quantity) { q.Significance = 0 }, "which 0008 requires"},
		{"no significance marker", func(q *Quantity) { q.Marker = "" }, "says nothing about how the count was arrived at"},
	} {
		t.Run(c.name, func(t *testing.T) {
			q := measured()
			c.change(&q)

			if _, err := json.Marshal(q); err == nil {
				t.Fatalf("a cell with %s encoded", c.name)
			} else if !strings.Contains(err.Error(), c.want) {
				t.Errorf("the refusal does not carry %q: %v", c.want, err)
			}
		})
	}
}

// Absent is the only route to a cell holding no value, and it refuses the pair
// at the point somebody builds it rather than at the point a bundle is written.
func TestAbsentRefusesAPairTheVocabularyDoesNotPermit(t *testing.T) {
	if _, err := Absent(Declined, NotConverged); err != nil {
		t.Fatalf("a permitted pair was refused: %v", err)
	}
	if _, err := Absent(Measured, ""); err == nil {
		t.Error("measured was accepted as an absence, which is the one state that is not one")
	}
	if _, err := Absent(Declined, Ambiguous); err == nil {
		t.Error("a refused reason was accepted under declined, which is the confusion 0011 names as the worst")
	}
}

// The same rules apply on the way in. A bundle written by something other than
// this code is read under them, or a hand edited artefact enters the comparison
// with a cell nothing judged.
func TestReadingACellAppliesTheSameRules(t *testing.T) {
	var q Quantity
	if err := json.Unmarshal([]byte(`{"state":"refused","reason":"ambiguous"}`), &q); err != nil {
		t.Fatalf("a well formed absent cell was refused on the way in: %v", err)
	}
	if q.State != Refused || q.Reason != Ambiguous {
		t.Fatalf("the cell read back as %+v", q)
	}

	for _, c := range []struct{ name, in, want string }{
		{"no state", `{"printed":"1.0"}`, "no state"},
		{"an absence with no reason", `{"state":"declined"}`, "carries no reason"},
		{"a field the model does not have", `{"state":"refused","reason":"ambiguous","note":"whatever"}`, "unknown field"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if err := json.Unmarshal([]byte(c.in), &q); err == nil {
				t.Fatalf("%s was read", c.name)
			} else if !strings.Contains(err.Error(), c.want) {
				t.Errorf("the refusal does not carry %q: %v", c.want, err)
			}
		})
	}
}

func TestACellSurvivesARoundTrip(t *testing.T) {
	b, err := json.Marshal(measured())
	if err != nil {
		t.Fatal(err)
	}
	var back Quantity
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back != measured() {
		t.Errorf("the cell came back as %+v rather than %+v", back, measured())
	}
}
