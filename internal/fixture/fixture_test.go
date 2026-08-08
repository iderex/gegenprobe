package fixture

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"
)

// The two fixtures below are the ones the convention exists for. One carries
// carriage returns, the other carries trailing spaces that a fixed format table
// depends on, and both are the bytes a checkout is most likely to rewrite. The
// assertions are on the exact byte sequence rather than on anything parsed from
// it, because a parsed result is exactly what survives the damage.
const (
	carriageReturns    = "  1  -1  1s   1  -0.5000000E+00\r\n  2   1  2p   3  -0.2500000E+00\r\n"
	trailingWhitespace = "LEVEL  ENERGY      J P   \n    1  0.00000     0 +   \n    2  1.23456     1 -   \n"

	// The digests are of the decoded bytes. They are written down so that a
	// clone on another platform can be checked against them from the shell,
	// without this suite and without a Go toolchain.
	carriageReturnsDigest    = "531656be1b247693330a84441e644b5eade9be63411dfe9ee2ebfa207e987301"
	trailingWhitespaceDigest = "aa2d4a8f6e44e1df75369147a42ce9cde77e1cbef27cdfb90307101021531c29"
)

func load(t *testing.T, name string) Fixture {
	t.Helper()
	f, err := Load(filepath.Join("testdata", name+Extension))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestTheCarriageReturnFixtureArrivesWithItsCarriageReturns(t *testing.T) {
	got := load(t, "carriage-return").Bytes

	if !bytes.Equal(got, []byte(carriageReturns)) {
		t.Fatalf("the fixture decoded to %q, want %q", got, carriageReturns)
	}
	if n := bytes.Count(got, []byte("\r\n")); n != 2 {
		t.Errorf("the decoded bytes hold %d CRLF pairs, want 2; a checkout that stripped them would pass every parsed assertion", n)
	}
}

func TestTheTrailingWhitespaceFixtureKeepsEveryTrailingSpace(t *testing.T) {
	got := load(t, "trailing-whitespace").Bytes

	if !bytes.Equal(got, []byte(trailingWhitespace)) {
		t.Fatalf("the fixture decoded to %q, want %q", got, trailingWhitespace)
	}
	for i, line := range strings.Split(strings.TrimSuffix(string(got), "\n"), "\n") {
		if !strings.HasSuffix(line, "   ") {
			t.Errorf("line %d is %q, and the three trailing spaces a fixed width table depends on are gone", i+1, line)
		}
	}
}

// The digests are what a clone on another platform is compared against, so they
// have to be the digests of these bytes and not of something near them.
func TestTheRecordedDigestsAreTheDigestsOfTheseBytes(t *testing.T) {
	for _, c := range []struct{ name, want string }{
		{"carriage-return", carriageReturnsDigest},
		{"trailing-whitespace", trailingWhitespaceDigest},
	} {
		t.Run(c.name, func(t *testing.T) {
			sum := sha256.Sum256(load(t, c.name).Bytes)
			if got := hex.EncodeToString(sum[:]); got != c.want {
				t.Errorf("the decoded bytes hash to %s, and the recorded digest is %s", got, c.want)
			}
		})
	}
}

func TestTheProvenanceNoteIsRead(t *testing.T) {
	f := load(t, "carriage-return")
	if f.Code == "" || f.Version == "" || f.Case == "" || f.Kept == "" {
		t.Fatalf("a field of the provenance note came back empty: %+v", f)
	}
	if !strings.Contains(f.Kept, "CRLF") {
		t.Errorf("Kept does not say what was kept: %q", f.Kept)
	}
}

// However the file itself is wrapped or line ended, the bytes it holds are the
// same. This is the property the convention is chosen for, so it is asserted
// rather than described: the same payload is rewrapped, indented, and given
// carriage returns, and all four decode identically.
func TestHowTheFileIsWrappedCannotReachTheBytes(t *testing.T) {
	want := load(t, "carriage-return").Bytes
	header := "Code: hand-written\nVersion: not applicable\nCase: not applicable\nKept: everything\n\n"
	payload := Encode(want)

	for _, c := range []struct{ name, body string }{
		{"as written", header + payload},
		{"one long line", header + strings.ReplaceAll(payload, "\n", "")},
		{"carriage returns everywhere", strings.ReplaceAll(header+payload, "\n", "\r\n")},
		{"indented", header + "    " + strings.ReplaceAll(payload, "\n", "\n    ")},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, problems := Parse([]byte(c.body))
			if len(problems) > 0 {
				t.Fatalf("%v", problems)
			}
			if !bytes.Equal(f.Bytes, want) {
				t.Errorf("decoded to %q, want %q", f.Bytes, want)
			}
		})
	}
}

// good is the passing fixture body every case below changes one line of.
func good() []string {
	return []string{
		"Code: GRASP2018",
		"Version: 2018.1",
		"Case: hydrogen-like-1s2p",
		"Kept: the first two lines of the level table",
		"",
		"aGVsbG8K",
		"",
	}
}

func body(lines []string) []byte { return []byte(strings.Join(lines, "\n")) }

func TestTheGoodBodyParses(t *testing.T) {
	f, problems := Parse(body(good()))
	if len(problems) > 0 {
		t.Fatalf("the passing body was refused: %v", problems)
	}
	if string(f.Bytes) != "hello\n" {
		t.Errorf("decoded to %q", f.Bytes)
	}
}

// Each case is the passing body with one line changed, removed or added, which
// is the mistake somebody actually makes rather than a body mangled beyond
// recognition.
func TestEachWayAFixtureIsWrongIsRefusedAndSaidOutLoud(t *testing.T) {
	for _, c := range []struct {
		name   string
		mutate func(lines []string) []string
		says   string
	}{
		{"a missing field", func(l []string) []string { return append(l[:1:1], l[2:]...) }, "missing Version"},
		{"an empty field", func(l []string) []string { l[1] = "Version:"; return l }, "Version is empty"},
		{"a field given twice", func(l []string) []string { l[2] = "Version: 2018.2"; return l }, "given twice"},
		{"a misspelt field", func(l []string) []string { l[1] = "Verison: 2018.1"; return l }, "not a fixture field"},
		{"a line that is not a field", func(l []string) []string { l[1] = "2018.1"; return l }, "is not a Field: value line"},
		{"no blank line", func(l []string) []string { return append(l[:4:4], l[5:]...) }, "no blank line"},
		{"a payload that is not base64", func(l []string) []string { l[5] = "not base64!"; return l }, "not base64"},
		{"an empty payload", func(l []string) []string { l[5] = ""; return l }, "holds no bytes at all"},
	} {
		t.Run(c.name, func(t *testing.T) {
			lines := c.mutate(good())

			_, problems := Parse(body(lines))

			if len(problems) == 0 {
				t.Fatalf("%s was accepted", c.name)
			}
			if !strings.Contains(strings.Join(problems, "\n"), c.says) {
				t.Errorf("the refusal does not say why:\n%s", strings.Join(problems, "\n"))
			}
			if len(lines) > len(good())+1 {
				t.Errorf("the case changed %d lines against the passing body's %d", len(lines), len(good()))
			}
		})
	}
}

func TestEncodeRoundTripsAnyBytes(t *testing.T) {
	for _, raw := range []string{"", "\r\n", "a", trailingWhitespace, carriageReturns, "\x00\x01\xff"} {
		payload := Encode([]byte(raw))
		f, problems := Parse([]byte("Code: c\nVersion: v\nCase: k\nKept: all\n\n" + payload))
		if raw == "" {
			if len(problems) == 0 {
				t.Error("a fixture holding no bytes was accepted")
			}
			continue
		}
		if len(problems) > 0 {
			t.Fatalf("%q: %v", raw, problems)
		}
		if string(f.Bytes) != raw {
			t.Errorf("round trip of %q gave %q", raw, f.Bytes)
		}
	}
}
