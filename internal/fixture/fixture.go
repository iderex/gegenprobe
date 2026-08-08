// Package fixture reads the recorded bytes this repository's tests are run
// against, and it is the only thing that reads them.
//
// A fixture is stored encoded rather than raw, because raw is not safe here. The
// readers are tested against output from fixed format Fortran programs, where
// column positions carry meaning, trailing spaces are significant and some files
// arrive with carriage returns. A checkout can rewrite every one of those bytes:
// a clone made with core.autocrlf set turns each stored newline into a carriage
// return and a newline in the working tree, and an editor saving a file it
// opened does the same thing in the other direction. Either way the fixture
// stops testing what it was collected to test, and it does so silently.
//
// So a fixture file holds a header naming where the bytes came from, a blank
// line, and the bytes in base64. Every whitespace byte in the payload is
// discarded before decoding, which is what makes the decoded bytes independent
// of how the file itself was wrapped, indented or line ended on the way in or
// out of git. docs/fixtures.md is where the convention is written down and
// where adding one is described.
package fixture

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Extension is what a fixture file is called. A file under a testdata directory
// with any other name is refused by the gate, because a reader cannot tell a
// raw fixture from an encoded one by looking and neither can a contributor.
const Extension = ".fixture"

// Fixture is one recorded artefact and the note saying where it came from. The
// note is not decoration: two codes built differently produce different bytes
// for the same physics, so a fixture without its origin proves something about
// an unknown program.
type Fixture struct {
	// Code is the program the bytes came from, or "hand-written" where they
	// were not produced by one.
	Code string
	// Version is the version of that program.
	Version string
	// Case is what it was asked to compute.
	Case string
	// Kept says how much of the original file was retained, so that a reader
	// meeting a truncated table knows it was truncated here rather than there.
	Kept string
	// Bytes is the recorded artefact itself, exactly as it was collected.
	Bytes []byte
}

// fields are the header fields a fixture carries, all four required and none
// other permitted. An unknown field is refused rather than ignored, because a
// misspelt one is otherwise a provenance note that silently is not there.
var fields = []string{"Code", "Version", "Case", "Kept"}

// Load reads one fixture file. The problems it reports are the same ones the
// gate leg reports, from the same function, so a fixture that loads in a test
// is a fixture the gate accepts and there is no second opinion.
func Load(path string) (Fixture, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Fixture{}, err
	}
	f, problems := Parse(body)
	if len(problems) > 0 {
		return Fixture{}, fmt.Errorf("%s:\n  %s", filepath.ToSlash(path), strings.Join(problems, "\n  "))
	}
	return f, nil
}

// Parse reads a fixture out of the bytes of a fixture file and says everything
// wrong with it rather than stopping at the first thing, so that somebody
// adding one is not led through the rules one failure at a time.
//
// Carriage returns in the file are normalised before anything is read. They are
// a property of the checkout rather than of the fixture, which is the whole
// reason the payload is encoded.
func Parse(body []byte) (Fixture, []string) {
	text := strings.ReplaceAll(string(body), "\r\n", "\n")
	header, payload, separated := strings.Cut(text, "\n\n")
	if !separated {
		return Fixture{}, []string{"there is no blank line between the header and the payload, so where the note ends and the bytes begin is not decidable"}
	}

	var problems []string
	seen := map[string]string{}
	for i, line := range strings.Split(header, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			problems = append(problems, fmt.Sprintf("header line %d, %q, is not a Field: value line", i+1, line))
			continue
		}
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if !known(name) {
			problems = append(problems, fmt.Sprintf("header line %d names %q, which is not a fixture field; the fields are %s", i+1, name, strings.Join(fields, ", ")))
			continue
		}
		if _, already := seen[name]; already {
			problems = append(problems, fmt.Sprintf("%s is given twice, so which value is the provenance is not decidable", name))
			continue
		}
		if value == "" {
			problems = append(problems, fmt.Sprintf("%s is empty; a field nobody filled in is worse than an absent one, because it reads as answered", name))
			continue
		}
		seen[name] = value
	}

	var missing []string
	for _, name := range fields {
		if _, ok := seen[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		problems = append(problems, "the provenance note is missing "+strings.Join(missing, ", "))
	}

	decoded, problem := decode(payload)
	if problem != "" {
		problems = append(problems, problem)
	}

	if len(problems) > 0 {
		return Fixture{}, problems
	}
	return Fixture{
		Code:    seen["Code"],
		Version: seen["Version"],
		Case:    seen["Case"],
		Kept:    seen["Kept"],
		Bytes:   decoded,
	}, nil
}

// decode strips every whitespace byte from the payload before decoding it. That
// one line is what the convention buys: how the file was wrapped or line ended
// cannot reach the bytes, so a checkout on any platform decodes to the same
// artefact.
func decode(payload string) ([]byte, string) {
	stripped := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, payload)
	if stripped == "" {
		return nil, "the payload is empty, so this fixture holds no bytes at all"
	}
	decoded, err := base64.StdEncoding.DecodeString(stripped)
	if err != nil {
		return nil, "the payload is not base64: " + err.Error()
	}
	return decoded, ""
}

// Encode renders bytes as a payload the convention accepts, wrapped so that a
// fixture stays readable in a diff. It exists so that the way a fixture is
// written is the way the reader expects, rather than something a contributor
// reproduces by hand.
func Encode(raw []byte) string {
	encoded := base64.StdEncoding.EncodeToString(raw)
	const width = 76
	var b strings.Builder
	for len(encoded) > width {
		b.WriteString(encoded[:width])
		b.WriteByte('\n')
		encoded = encoded[width:]
	}
	b.WriteString(encoded)
	b.WriteByte('\n')
	return b.String()
}

func known(name string) bool {
	for _, f := range fields {
		if f == name {
			return true
		}
	}
	return false
}
