package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// recorded builds a leg that reports what it was told to report and records that
// it ran, so a test can assert on which legs the sequencing reached.
func recorded(name string, o outcome, ran *[]string) leg {
	return leg{
		name:    name,
		subject: "fixture",
		run: func(string) outcome {
			*ran = append(*ran, name)
			return o
		},
	}
}

func TestEveryLegHasANameAndTheSetIsNotEmpty(t *testing.T) {
	ls := legs()
	if len(ls) == 0 {
		t.Fatal("the command knows about no legs, so a green run would mean nothing")
	}
	seen := map[string]bool{}
	for i, l := range ls {
		if strings.TrimSpace(l.name) == "" {
			t.Errorf("leg %d has no name; a leg without one cannot be reported", i)
		}
		if strings.TrimSpace(l.subject) == "" {
			t.Errorf("leg %q states no subject, so a pass from it says nothing specific", l.name)
		}
		if l.run == nil {
			t.Errorf("leg %q has nothing to run", l.name)
		}
		if seen[l.name] {
			t.Errorf("two legs are named %q, so a verdict cannot be attributed to either", l.name)
		}
		seen[l.name] = true
	}
}

func TestRunStopsAtTheFirstFailureAndNamesWhatDidNotRun(t *testing.T) {
	var ran []string
	var out bytes.Buffer

	code := run(&out, ".", []leg{
		recorded("first", pass(), &ran),
		recorded("second", fail("the fixture said no"), &ran),
		recorded("third", pass(), &ran),
	})

	if code == 0 {
		t.Error("a failing leg left the exit status at zero")
	}
	if got := strings.Join(ran, ","); got != "first,second" {
		t.Errorf("legs that ran: %q; the run did not stop at the failure", got)
	}
	for _, want := range []string{"FAILED at leg 2 of 3, second", "the fixture said no", "did not run: third"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the account of the run does not carry %q:\n%s", want, out.String())
		}
	}
}

func TestASkippedLegSaysWhatWasMissingAndTheRunGoesOn(t *testing.T) {
	var ran []string
	var out bytes.Buffer

	code := run(&out, ".", []leg{
		recorded("absent tool", skip("the analyser is not installed"), &ran),
		recorded("after", pass(), &ran),
	})

	if code != 0 {
		t.Errorf("a skipped leg failed the run; exit status %d", code)
	}
	if got := strings.Join(ran, ","); got != "absent tool,after" {
		t.Errorf("legs that ran: %q; a skip stopped the run", got)
	}
	for _, want := range []string{"skip", "the analyser is not installed", "examined nothing: absent tool", "covered less than the whole set"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("a skipped leg passed without saying so: %q missing from:\n%s", want, out.String())
		}
	}
}

func TestTheRunNamesEveryLegItKnowsAbout(t *testing.T) {
	var out bytes.Buffer
	var ran []string
	ls := []leg{recorded("alpha", pass(), &ran), recorded("beta", pass(), &ran)}

	run(&out, ".", ls)

	for _, l := range ls {
		if !strings.Contains(out.String(), l.name) {
			t.Errorf("leg %q is not named in the account of the run:\n%s", l.name, out.String())
		}
	}
}

// writeGo puts a Go file in a temporary tree the format leg can walk. The bytes
// are written exactly as given, because the line endings are what two of these
// cases are about.
func writeGo(t *testing.T, dir, name, src string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

const formatted = "package x\n\nfunc f() int {\n\treturn 1\n}\n"

func TestFormatLegRefusesAFileThatIsNotFormatted(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, dir, "bad.go", "package x\n\nfunc f() int {\n        return 1\n}\n")

	o := formatLeg(dir)

	if o.verdict != failed {
		t.Fatalf("a file indented with spaces passed the format leg: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "bad.go") {
		t.Errorf("the failure does not name the file:\n%s", o.detail)
	}
	if !strings.Contains(o.detail, "gofmt -w") {
		t.Errorf("the failure does not say how to repair it:\n%s", o.detail)
	}
}

func TestFormatLegAcceptsAFormattedFile(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, dir, "good.go", formatted)

	if o := formatLeg(dir); o.verdict != passed {
		t.Fatalf("a gofmt formatted file was refused: %s", o.detail)
	}
}

// A checkout on Windows can hold every file with carriage returns, and that is
// not a formatting difference. The leg passes and says it normalised, so the
// pass is not silent about what it did.
func TestFormatLegNormalisesCarriageReturnsAndSaysSo(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, dir, "crlf.go", strings.ReplaceAll(formatted, "\n", "\r\n"))

	o := formatLeg(dir)

	if o.verdict != passed {
		t.Fatalf("a formatted file with CRLF endings was refused: %s", o.detail)
	}
	if !strings.Contains(o.detail, "CRLF") {
		t.Errorf("the leg normalised line endings without saying so: %q", o.detail)
	}
}

func TestFormatLegRefusesAFileItCannotParse(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, dir, "broken.go", "package x\n\nfunc f( {\n")

	o := formatLeg(dir)

	if o.verdict != failed {
		t.Fatal("a file that does not parse passed the format leg")
	}
	if !strings.Contains(o.detail, "broken.go") {
		t.Errorf("the failure does not name the file:\n%s", o.detail)
	}
}
