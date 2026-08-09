package model

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/iderex/gegenprobe/internal/fixture"
)

// members is every type this package writes into a bundle. The reflection tests
// below walk it, so a type added to the bundle without being added here is
// outside what they assert, and that is the one thing about them worth knowing.
func members() []any { return []any{Manifest{}, Run{}, Result{}} }

// The condition the issue asks for by name. Every number in this model that
// carries physical meaning is a Quantity, which holds its unit and its
// significance count beside the value, and there is no other float anywhere in
// the model for one to escape into.
//
// Stated as a rule over the type set rather than as a list of fields, because a
// list would be a second copy of the model and would go stale on the first field
// somebody adds.
func TestEveryNumberInTheModelCarriesAUnitAndASignificanceCount(t *testing.T) {
	quantity := reflect.TypeOf(Quantity{})

	if _, ok := quantity.FieldByName("Unit"); !ok {
		t.Fatal("Quantity has no Unit field, so the rule below asserts nothing")
	}
	if _, ok := quantity.FieldByName("Significance"); !ok {
		t.Fatal("Quantity has no Significance field, so the rule below asserts nothing")
	}

	var floats, loose []string
	for _, m := range members() {
		walk(reflect.TypeOf(m), reflect.TypeOf(m).Name(), map[reflect.Type]bool{}, func(path string, owner reflect.Type, f reflect.StructField) {
			if f.Type.Kind() != reflect.Float64 && f.Type.Kind() != reflect.Float32 {
				return
			}
			floats = append(floats, path)
			if owner != quantity {
				loose = append(loose, path+", declared on "+owner.Name())
			}
		})
	}

	if len(floats) == 0 {
		t.Fatal("no float was found anywhere in the model, so this test walked nothing")
	}
	sort.Strings(loose)
	for _, path := range loose {
		t.Errorf("%s is a float outside a Quantity, so it carries neither a unit nor a significance count", path)
	}
}

// The rule above says nothing unless a cell is actually reached from the tables
// a code's output lands in, so the reachability is asserted rather than assumed.
func TestTheParticipantTablesReachACell(t *testing.T) {
	found := 0
	walk(reflect.TypeOf(Result{}), "Result", map[reflect.Type]bool{}, func(path string, owner reflect.Type, f reflect.StructField) {
		if f.Type == reflect.TypeOf(Quantity{}) {
			found++
		}
	})

	if found == 0 {
		t.Fatal("a participant's tables reach no cell, so nothing in them carries a state")
	}
}

// The observed energies exist and are populated by nothing here. Both halves
// matter: the fields have to be in the type so that the fit component does not
// force a format bump later, and they have to be absent with a reason rather
// than zero, because zero is a physically meaningful energy.
func TestTheObservedEnergiesExistAndAreUnpopulated(t *testing.T) {
	observed, ok := reflect.TypeOf(Level{}).FieldByName("Observed")
	if !ok {
		t.Fatal("Level carries no observed energy, so the fit component would need a format bump to add one")
	}
	for _, name := range []string{"Energy", "Uncertainty", "Source"} {
		if _, ok := observed.Type.FieldByName(name); !ok {
			t.Errorf("the observed block carries no %s", name)
		}
	}

	unpopulated, err := Absent(NotRequested, QuantityNotRequested)
	if err != nil {
		t.Fatal(err)
	}
	level := Level{Observed: Observed{Energy: unpopulated, Uncertainty: unpopulated}}
	b, err := json.Marshal(level.Observed)
	if err != nil {
		t.Fatalf("the shape this release writes for an observed energy was refused: %v", err)
	}
	if !strings.Contains(string(b), `"state":"not-requested"`) {
		t.Errorf("the observed energy this release writes is not marked absent:\n%s", b)
	}
}

// A stored bundle from a format this build does not read, and the message it
// gets. The fixture is one field away from a manifest this build reads, so what
// is refused is the version and nothing else about it.
func TestABundleUnderAnOlderFormatIsRefusedNamingBothVersions(t *testing.T) {
	f, err := fixture.Load(filepath.Join("testdata", "a-bundle-written-under-an-older-format"+fixture.Extension))
	if err != nil {
		t.Fatal(err)
	}

	_, err = ReadManifest(f.Bytes)

	if err == nil {
		t.Fatal("a bundle written under format 0 was read by a build that writes format 1")
	}
	for _, want := range []string{"bundle format 0", "this build reads 1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}

	// The same bytes with the version corrected are read, which is what makes
	// the refusal above about the version.
	corrected := strings.Replace(string(f.Bytes), `"bundle-format":0`, `"bundle-format":1`, 1)
	m, err := ReadManifest([]byte(corrected))
	if err != nil {
		t.Fatalf("the same manifest at this build's format was refused: %v", err)
	}
	if m.Format != Format || len(m.Members) != 4 {
		t.Errorf("the manifest read back as format %d with %d member(s)", m.Format, len(m.Members))
	}
}

func TestABundleUnderANewerFormatIsRefusedAndNotReadPartially(t *testing.T) {
	_, err := ReadManifest([]byte(`{"bundle-format":2,"case-id":"x"}`))

	if err == nil {
		t.Fatal("a bundle from a newer format was read")
	}
	for _, want := range []string{"bundle format 2", "this build reads 1", "is not read partially"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not carry %q: %v", want, err)
		}
	}
}

func TestAManifestWithNoFormatFieldIsRefused(t *testing.T) {
	_, err := ReadManifest([]byte(`{"case-id":"x"}`))

	if err == nil {
		t.Fatal("a manifest declaring no format was read")
	}
	if !strings.Contains(err.Error(), "no `bundle-format` field") {
		t.Errorf("the refusal does not say what is missing: %v", err)
	}
}

// walk visits every field of a struct type, following slices and nested
// structs, and reports the path a reader would follow to reach it.
func walk(t reflect.Type, path string, seen map[reflect.Type]bool, do func(string, reflect.Type, reflect.StructField)) {
	for t.Kind() == reflect.Slice || t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || seen[t] {
		return
	}
	seen[t] = true
	defer delete(seen, t)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		do(path+"."+f.Name, t, f)
		walk(f.Type, path+"."+f.Name, seen, do)
	}
}
