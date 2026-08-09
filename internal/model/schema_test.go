package model

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// schemaDefs decodes the generated schema once per test that needs it, so the
// assertions below are about the bytes this package would write rather than
// about the structures it holds in memory.
func schemaDefs(t *testing.T) (map[string]any, []byte) {
	t.Helper()
	b, err := Schema()
	if err != nil {
		t.Fatalf("the schema did not generate: %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		t.Fatalf("the generated schema is not JSON: %v", err)
	}
	defs, ok := root["$defs"].(map[string]any)
	if !ok {
		t.Fatalf("the generated schema holds no definitions:\n%s", b)
	}
	return defs, b
}

// The conformance condition, and it comes from 0004, which makes this model the
// thing every reader and every later stage agrees through, and 0007, which makes
// the bundle the artefact a consumer reads without this repository in hand. A
// schema that describes a field the types do not carry, or misses one they do,
// breaks the agreement in the direction nobody notices until a consumer meets a
// real bundle.
//
// Both directions are asserted over every type a bundle carries and not only
// over the three files at the top. A generator that dropped a nested type would
// pass a check that stopped at the members, and the nested types are where all
// the physics is.
func TestTheSchemaAndTheTypesCoverEachOther(t *testing.T) {
	defs, raw := schemaDefs(t)
	carried := typesABundleCarries()

	for name := range defs {
		if _, ok := carried[name]; !ok {
			t.Errorf("the schema defines %s and no type a bundle carries has that name", name)
		}
	}

	for name, typ := range carried {
		def, ok := defs[name].(map[string]any)
		if !ok {
			t.Errorf("the schema describes no %s, which is a type the bundle carries", name)
			continue
		}
		properties, ok := def["properties"].(map[string]any)
		if !ok {
			t.Errorf("the definition of %s holds no properties:\n%s", name, raw)
			continue
		}

		for i := 0; i < typ.NumField(); i++ {
			field, _ := jsonName(typ.Field(i))
			if _, described := properties[field]; !described {
				t.Errorf("%s carries the field %q and the schema describes it nowhere", name, field)
			}
		}
		for field := range properties {
			if !fieldNamed(typ, field) {
				t.Errorf("the schema describes %s.%q and the type has no such field", name, field)
			}
		}
	}
}

// A check that walked nothing would pass every assertion above, so what it
// walked is asserted rather than assumed: the type set has to reach past the
// three files into the tables inside them.
func TestTheConformanceCheckReachesTheTablesAndNotOnlyTheFiles(t *testing.T) {
	carried := typesABundleCarries()

	for _, want := range []string{"Manifest", "Run", "Result", "Level", "Transition", "Quantity", "Observed", "Strength", "Step", "Variable", "VariableStep", "Member", "WrittenBy"} {
		if _, ok := carried[want]; !ok {
			t.Errorf("%s is a type a bundle carries and the conformance check does not reach it", want)
		}
	}
}

// typesABundleCarries is every struct type reachable from a bundle member,
// derived from the types themselves rather than listed. A list here would be a
// second copy of the model, and it would go stale on the first type somebody
// adds, which is the drift this whole check exists against.
func typesABundleCarries() map[string]reflect.Type {
	out := map[string]reflect.Type{}
	for _, m := range members() {
		collect(reflect.TypeOf(m), out)
	}
	return out
}

func collect(t reflect.Type, into map[string]reflect.Type) {
	for t.Kind() == reflect.Slice || t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	if _, seen := into[t.Name()]; seen {
		return
	}
	into[t.Name()] = t
	for i := 0; i < t.NumField(); i++ {
		if f := t.Field(i); f.IsExported() {
			collect(f.Type, into)
		}
	}
}

// The enumerations are the part of the generated schema worth having. A
// consumer reading it learns which states, reasons and units are admissible,
// and it learns them from the same slices the code judges against.
func TestTheSchemaCarriesTheVocabulariesTheCodeJudgesAgainst(t *testing.T) {
	defs, raw := schemaDefs(t)

	quantity, ok := defs["Quantity"].(map[string]any)
	if !ok {
		t.Fatalf("the schema describes no Quantity:\n%s", raw)
	}
	properties := quantity["properties"].(map[string]any)

	for _, c := range []struct {
		field string
		want  []string
	}{
		{"state", stateNames()},
		{"unit", unitNames(units)},
		{"significance-marker", markerNames()},
	} {
		described, ok := properties[c.field].(map[string]any)
		if !ok {
			t.Errorf("the schema describes no %q on a cell", c.field)
			continue
		}
		values, ok := described["enum"].([]any)
		if !ok {
			t.Errorf("%q is described without an enumeration, so a consumer learns nothing about what it may hold", c.field)
			continue
		}
		if len(values) != len(c.want) {
			t.Errorf("%q enumerates %d value(s) against the %d the code judges against", c.field, len(values), len(c.want))
			continue
		}
		for i, v := range values {
			if v != c.want[i] {
				t.Errorf("%q enumerates %v at position %d where the code holds %q", c.field, v, i, c.want[i])
			}
		}
	}

	reason := properties["reason"].(map[string]any)
	values := reason["enum"].([]any)
	if len(values) != 13 {
		t.Errorf("the reason vocabulary enumerates %d value(s); 0011 names thirteen across the three absent states", len(values))
	}
}

// A schema nobody can hold beside a bundle file is a schema in a different form
// from the thing it describes, so it is written in the same canonical form 0007
// requires of the bundle itself.
func TestTheSchemaIsCanonicalJSON(t *testing.T) {
	_, raw := schemaDefs(t)

	if strings.HasSuffix(string(raw), "\n") {
		t.Error("the schema ends in a newline, which the canonical form of 0007 does not carry")
	}
	if strings.ContainsAny(string(raw), "\n\t") {
		t.Error("the schema holds insignificant whitespace")
	}
	if !strings.Contains(string(raw), `"$id":"`+SchemaID+`"`) {
		t.Error("the schema does not name itself")
	}
	if !strings.Contains(string(raw), `"bundle-format":1`) {
		t.Error("the schema does not say which bundle format it describes, so a consumer cannot tell which one it has")
	}
}

// Generating twice produces the same bytes. A schema that moved between two
// runs of the same build would be undiffable, which is the property 0007 asks
// of everything in a bundle.
func TestGeneratingTheSchemaTwiceProducesTheSameBytes(t *testing.T) {
	first, err := Schema()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Schema()
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Error("two generations of the schema produced different bytes")
	}
}

func fieldNamed(t reflect.Type, want string) bool {
	for i := 0; i < t.NumField(); i++ {
		if name, _ := jsonName(t.Field(i)); name == want {
			return true
		}
	}
	return false
}
