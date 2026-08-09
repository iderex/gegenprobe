package model

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ReadManifest reads a bundle manifest and refuses one written under a format
// version this build does not know, naming both versions.
//
// It names both because one alone is unusable. A message saying only the
// version found leaves the reader guessing what this build wanted; a message
// saying only what this build wanted leaves them guessing what they have. The
// pair tells them whether to fetch a newer harness or an older one, which is the
// decision they are actually making.
//
// It does not read a newer bundle partially, for the reason 0002 gives for a
// newer case: a document a reader half understands is one it has understood
// wrongly and cannot know it. Every released format stays readable indefinitely,
// so a version below this one is read where a reader for it exists, and only a
// version nothing here has ever written is refused outright.
func ReadManifest(b []byte) (Manifest, error) {
	var probe struct {
		Format *int `json:"bundle-format"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return Manifest{}, fmt.Errorf("bundle manifest: %w", err)
	}
	if probe.Format == nil {
		return Manifest{}, fmt.Errorf("bundle manifest: no `bundle-format` field, so nothing says which format this bundle was written under; this build writes and reads %d", Format)
	}
	if *probe.Format != Format {
		return Manifest{}, fmt.Errorf("bundle manifest: written under bundle format %d and this build reads %d; %s",
			*probe.Format, Format, direction(*probe.Format))
	}

	var m Manifest
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("bundle manifest: %w", err)
	}
	return m, nil
}

// direction says which way the reader has to move, because the two cases have
// opposite answers and a message that treats them alike sends half its readers
// the wrong way.
func direction(found int) string {
	if found > Format {
		return "the bundle is newer than this build and is not read partially, so read it with a harness that knows that format"
	}
	return "no reader for that format is in this build, and every released format stays readable, so this is a bundle from a format that was never released"
}
