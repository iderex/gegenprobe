// Package golden compares an artefact a test produced against a copy of it
// committed to this repository, and rewrites that copy where the suite was run
// with the update flag.
//
// Most of what this project will assert about is a table: levels, transitions,
// an agreement map, an index over the decision records. A comparison that
// reports the first differing byte of a table is unreadable at the size those
// reach, and it is unreadable in the direction that matters, because it says
// where the bytes parted company rather than which row is wrong. So the unit
// here is a record rather than a byte, and the failure names the record.
//
// A golden file is committed and reviewed like any other file. The update flag
// exists so that a deliberate change to a producer is one command rather than
// an afternoon of transcription; it does not exist so that a red run becomes a
// green one. What it rewrites still arrives in a diff somebody reads.
package golden

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// update is the flag the suite is run with to rewrite what it compares against.
//
// It is declared by this package rather than by each caller so that there is one
// spelling of it. The cost of that is worth stating: a flag belongs to the test
// binary that defines it, so `go test ./... -update` is refused by the binaries
// of packages that hold no golden. The update run names the packages that do.
var update = flag.Bool("update", false, "rewrite the golden files this suite compares against")

// linkTarget reduces a Markdown link to the text a reader sees, so a record
// keyed by a linked number is keyed by the number and not by the file it points
// at. A key that moved because a file was renamed would report every record as
// changed.
var linkTarget = regexp.MustCompile(`^\[([^\]]*)\]\([^)]*\)$`)

// Difference is one record the two sides do not agree about. Key is what the
// record is known by, and it is empty only for a line the artefact carries
// outside any table, which is the case this package keys worst and says so.
type Difference struct {
	Key  string
	Why  string
	Want string
	Got  string
}

// The three things that can be wrong with a record, in the words the report
// uses. They are separate because "the golden holds a record nothing produced"
// and "a producer wrote a record the golden does not hold" are opposite
// statements, and a comparison that collapsed them would report a renumbered
// record as two unrelated faults.
const (
	whyDiffers     = "the golden and the producer disagree about it"
	whyNotProduced = "the golden holds it and nothing produced it"
	whyNotInGolden = "it was produced and the golden does not hold it"
	whyOutside     = "the text outside the table differs"
)

// Compare returns what the golden and the produced bytes disagree about, in the
// golden's own order, or nothing where they agree.
//
// Both sides are read after normalising CRLF to LF. What the bytes in a
// checkout are is decided by .gitattributes and is not this comparison's
// subject; a run that reported every record of every artefact as changed
// because of a line ending would say nothing about any producer.
func Compare(golden, produced []byte) []Difference {
	wantKeyed, wantLoose := records(golden)
	gotKeyed, gotLoose := records(produced)

	var found []Difference

	for _, key := range wantKeyed.order {
		got, present := gotKeyed.text[key]
		switch {
		case !present:
			found = append(found, Difference{Key: key, Why: whyNotProduced, Want: wantKeyed.text[key]})
		case got != wantKeyed.text[key]:
			found = append(found, Difference{Key: key, Why: whyDiffers, Want: wantKeyed.text[key], Got: got})
		}
	}
	for _, key := range gotKeyed.order {
		if _, present := wantKeyed.text[key]; !present {
			found = append(found, Difference{Key: key, Why: whyNotInGolden, Got: gotKeyed.text[key]})
		}
	}

	if d, differs := looseDifference(wantLoose, gotLoose); differs {
		found = append(found, d)
	}
	return found
}

// keyed is the records of one side that carry a key, in the order they were
// written.
type keyed struct {
	order []string
	text  map[string]string
}

// records splits an artefact into the records that carry a key and the lines
// that do not. A record is a line, which is what every artefact this repository
// writes is made of, and a key is what a reader would call the row: the first
// cell of a table row, with a link reduced to the text inside it.
//
// A line outside a table carries no key. Those are compared in order and
// reported by quoting them, which is weaker than a keyed comparison and is the
// bound this package has. It is the reason the failure this is written for is
// stated about a table.
func records(artefact []byte) (keyed, []string) {
	k := keyed{text: map[string]string{}}
	var loose []string

	for _, line := range strings.Split(string(bytes.ReplaceAll(artefact, []byte("\r\n"), []byte("\n"))), "\n") {
		key := keyOf(line)
		if key == "" {
			loose = append(loose, line)
			continue
		}
		if _, seen := k.text[key]; seen {
			// A repeated key is not a key. Both lines go to the ordered
			// comparison rather than one of them silently replacing the other.
			loose = append(loose, k.text[key], line)
			continue
		}
		k.order = append(k.order, key)
		k.text[key] = line
	}
	return k, loose
}

// keyOf returns what a line is known by, or the empty string where it is not a
// table row this can key. A separator row keys nothing, which is what keeps a
// table's own scaffolding out of the record set.
func keyOf(line string) string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") {
		return ""
	}
	first, _, found := strings.Cut(strings.TrimPrefix(trimmed, "|"), "|")
	if !found {
		return ""
	}
	key := strings.TrimSpace(first)
	if m := linkTarget.FindStringSubmatch(key); m != nil {
		key = strings.TrimSpace(m[1])
	}
	if strings.Trim(key, "-: ") == "" {
		return ""
	}
	return key
}

// looseDifference reports the first line outside any table the two sides do not
// agree about. One difference rather than all of them, because these lines are
// prose and a reader repairs the first and regenerates.
func looseDifference(want, got []string) (Difference, bool) {
	for i := range max(len(want), len(got)) {
		w, g := "", ""
		if i < len(want) {
			w = want[i]
		}
		if i < len(got) {
			g = got[i]
		}
		if w != g {
			return Difference{Why: whyOutside, Want: w, Got: g}, true
		}
	}
	return Difference{}, false
}

// Report renders what a comparison found, naming the artefact it is about.
func Report(name string, found []Difference) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s is not what its producer writes, in %d place(s):\n", name, len(found))
	for _, d := range found {
		b.WriteString("\n")
		if d.Key == "" {
			fmt.Fprintf(&b, "%s\n", d.Why)
		} else {
			fmt.Fprintf(&b, "record %s: %s\n", d.Key, d.Why)
		}
		if d.Want != "" {
			fmt.Fprintf(&b, "    golden:   %s\n", d.Want)
		}
		if d.Got != "" {
			fmt.Fprintf(&b, "    produced: %s\n", d.Got)
		}
	}
	return b.String()
}

// Assert compares what a producer wrote against the golden at path, and fails
// the test naming the record where they differ. Where the suite was run with
// the update flag it rewrites the golden and asserts nothing, which is the only
// mode in which this writes anything at all.
func Assert(t testing.TB, path string, produced []byte) {
	t.Helper()

	if *update {
		if err := os.WriteFile(path, produced, 0o644); err != nil {
			t.Fatalf("the golden could not be rewritten: %v", err)
		}
		return
	}

	if err := check(path, produced); err != nil {
		t.Fatal(err)
	}
}

// check is the whole of the judgement, separated from the assertion so that the
// suite can drive the failing case. A helper whose failure path is only ever
// reached by a test that fails is a helper whose failure path nothing reads.
func check(path string, produced []byte) error {
	committed, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("the golden could not be read: %v\n\nWrite it by running this suite with -update, and read what it wrote", err)
	}
	if found := Compare(committed, produced); len(found) > 0 {
		return fmt.Errorf("%s\nRerun this package's suite with -update to rewrite it. A golden is committed and\nreviewed like any other file, so what that writes still arrives in a diff", Report(filepath.ToSlash(path), found))
	}
	return nil
}
