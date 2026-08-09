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

// The condition the issue asks for: the schema is generated from the types
// rather than maintained beside them, so every field a bundle carries is
// described and no described field is one the types do not have. Both
// directions are asserted, because a generator that dropped a field would pass
// one of them.
func TestTheSchemaAndTheTypesCoverEachOther(t *testing.T) {
	defs, raw := schemaDefs(t)

	for _, m := range members() {
		typ := reflect.TypeOf(m)
		def, ok := defs[typ.Name()].(map[string]any)
		if !ok {
			t.Errorf("the schema describes no %s, which is a file the bundle carries", typ.Name())
			continue
		}
		properties, ok := def["properties"].(map[string]any)
		if !ok {
			t.Errorf("the definition of %s holds no properties:\n%s", typ.Name(), raw)
			continue
		}

		for i := 0; i < typ.NumField(); i++ {
			name, _ := jsonName(typ.Field(i))
			if _, described := properties[name]; !described {
				t.Errorf("%s carries the field %q and the schema describes it nowhere", typ.Name(), name)
			}
		}
		for name := range properties {
			if !fieldNamed(typ, name) {
				t.Errorf("the schema describes %s.%q and the type has no such field", typ.Name(), name)
			}
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
