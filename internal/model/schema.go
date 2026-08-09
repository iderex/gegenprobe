package model

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// The schema is generated from the types rather than written beside them.
//
// A schema kept by hand drifts the moment somebody adds a field, and the drift
// is invisible until a consumer written against the schema meets a bundle
// written by the types. Generating it means the two cannot disagree: there is
// one description of a bundle and the other is derived from it.
//
// What the generator reads is the struct tags, which are the same tags the
// encoder reads, so a field the schema describes is a field a bundle carries
// under that name. What it cannot read is meaning. Whether a field named
// `energy` holds an energy is a judgement, and no reading of the tree makes it.

// SchemaID is what the generated schema calls itself.
const SchemaID = "https://github.com/iderex/gegenprobe/bundle"

// Schema returns the JSON Schema describing every file under `bundle/` that
// this package's types write, as canonical JSON: keys sorted, no insignificant
// whitespace, no trailing newline. That is the same form 0007 requires of the
// bundle files themselves, and using it here means the schema is diffable by
// the same means as everything beside it.
func Schema() ([]byte, error) {
	defs := map[string]any{}
	members := map[string]any{}

	// One entry per bundle member this package's types describe. The three
	// missing members, the case, the identification and the comparison, belong
	// to the milestones that decide their shape and are named in the schema as
	// absent rather than left for a reader to notice.
	for _, m := range []struct {
		path  string
		value any
	}{
		{"manifest.json", Manifest{}},
		{"run.json", Run{}},
		{"result/<participant>.json", Result{}},
	} {
		ref, err := define(defs, reflect.TypeOf(m.value))
		if err != nil {
			return nil, err
		}
		members[m.path] = ref
	}

	root := map[string]any{
		"$schema":       "https://json-schema.org/draft/2020-12/schema",
		"$id":           SchemaID,
		"bundle-format": Format,
		"members":       members,
		"$defs":         defs,
	}
	return canonicalJSON(root)
}

// define adds one struct type to the definitions and returns a reference to it.
// Types are keyed by their Go name, which is what makes the schema readable
// beside the source it came from.
func define(defs map[string]any, t reflect.Type) (map[string]any, error) {
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("schema: %s is not a struct, and only a struct is a member of a bundle", t)
	}
	name := t.Name()
	ref := map[string]any{"$ref": "#/$defs/" + name}
	if _, done := defs[name]; done {
		return ref, nil
	}
	// Reserve the name before walking the fields so a type reaching itself
	// terminates rather than recurring forever.
	defs[name] = map[string]any{}

	properties := map[string]any{}
	var required []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		key, omitempty := jsonName(f)
		if key == "" {
			continue
		}
		schema, err := describe(defs, f.Type)
		if err != nil {
			return nil, fmt.Errorf("schema: %s.%s: %w", t.Name(), f.Name, err)
		}
		properties[key] = schema
		if !omitempty {
			required = append(required, key)
		}
	}
	sort.Strings(required)

	defs[name] = map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}
	return ref, nil
}

// describe turns one field's type into its schema. A named string type with a
// vocabulary becomes an enumeration, which is the part of this generator worth
// having: the admissible values of a state, a reason or a unit are in the
// schema because they are in the source, and neither can move without the
// other.
func describe(defs map[string]any, t reflect.Type) (map[string]any, error) {
	if values, ok := vocabulary(t); ok {
		return map[string]any{"type": "string", "enum": values}, nil
	}
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}, nil
	case reflect.Bool:
		return map[string]any{"type": "boolean"}, nil
	case reflect.Int:
		return map[string]any{"type": "integer"}, nil
	case reflect.Float64:
		return map[string]any{"type": "number"}, nil
	case reflect.Slice:
		item, err := describe(defs, t.Elem())
		if err != nil {
			return nil, err
		}
		return map[string]any{"type": "array", "items": item}, nil
	case reflect.Struct:
		return define(defs, t)
	default:
		return nil, fmt.Errorf("%s is a kind this schema does not describe", t.Kind())
	}
}

// vocabularies is the one place a named string type's admissible values are
// stated for the schema, and each entry points at the same slice the code
// judges against. A type here whose values were listed a second time would be
// the drift this generator exists against, one level down.
func vocabulary(t reflect.Type) ([]string, bool) {
	switch t {
	case reflect.TypeOf(State("")):
		return stateNames(), true
	case reflect.TypeOf(Reason("")):
		var out []string
		for _, s := range states {
			out = append(out, reasonNames(s)...)
		}
		sort.Strings(out)
		return out, true
	case reflect.TypeOf(Unit("")):
		return unitNames(units), true
	case reflect.TypeOf(Marker("")):
		return markerNames(), true
	case reflect.TypeOf(Parity("")):
		return names(parities), true
	case reflect.TypeOf(Multipole("")):
		return names(multipoles), true
	case reflect.TypeOf(Gauge("")):
		return names(gauges), true
	case reflect.TypeOf(Medium("")):
		return names(media), true
	}
	return nil, false
}

func names[T ~string](in []T) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, string(v))
	}
	return out
}

// jsonName reads the tag the encoder reads, so the schema names a field exactly
// as a bundle carries it. A field tagged `-` is described by nothing because it
// is written by nothing.
func jsonName(f reflect.StructField) (string, bool) {
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", false
	}
	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		name = f.Name
	}
	omitempty := false
	for _, p := range parts[1:] {
		if p == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty
}

// canonicalJSON writes the form 0007 requires: sorted keys, no insignificant
// whitespace, no trailing newline. Go's encoder sorts a map's keys already, and
// this drops the newline it adds so that the bytes are the artefact rather than
// nearly it.
func canonicalJSON(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}
	return b, nil
}
