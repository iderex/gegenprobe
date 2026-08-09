package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iderex/gegenprobe/internal/golden"
)

// record writes a minimal file in the format 0000 fixes. The four body sections
// are not read by this command, so the fixtures carry only what it parses.
func writeRecord(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func good(number, title, status, extra string) string {
	return "# " + number + ". " + title + "\n\n" +
		"Number: " + number + "\n" +
		"Title: " + title + "\n" +
		"Status: " + status + "\n" +
		"Date: 2026-08-07\n" +
		extra +
		"\n## What was decided\n\nx\n"
}

func twoRecords(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeRecord(t, dir, "0000-decision-records.md", good("0000", "Decision records", "accepted", ""))
	writeRecord(t, dir, "0002-the-case-file.md", good("0002", "The case file", "accepted", ""))
	return dir
}

func TestWritesIndexInNumberOrder(t *testing.T) {
	dir := twoRecords(t)
	if err := run(dir, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "| [0000](0000-decision-records.md) | Decision records | accepted | 2026-08-07 |\n" +
		"| [0002](0002-the-case-file.md) | The case file | accepted | 2026-08-07 |\n"
	if !strings.HasSuffix(string(got), want) {
		t.Fatalf("index does not end with the two rows in number order:\n%s", got)
	}
	if strings.Contains(string(got), "README.md)") {
		t.Fatal("the index lists itself")
	}
}

func TestRunTwiceLeavesTheSameBytes(t *testing.T) {
	dir := twoRecords(t)
	if err := run(dir, false); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := run(dir, false); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("a second run changed the index")
	}
	if err := run(dir, true); err != nil {
		t.Fatalf("-check reported a current index as stale: %v", err)
	}
}

func TestCheckRefusesAStaleIndexAndWritesNothing(t *testing.T) {
	dir := twoRecords(t)
	if err := run(dir, false); err != nil {
		t.Fatal(err)
	}
	writeRecord(t, dir, "0004-a-third-one.md", good("0004", "A third one", "proposed", ""))

	before, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := run(dir, true); err == nil {
		t.Fatal("-check accepted an index missing a record")
	}
	after, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("-check wrote to the index")
	}
}

func TestCRLFOnDiskIsNotAStaleIndex(t *testing.T) {
	dir := twoRecords(t)
	if err := run(dir, false); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "README.md")
	lf, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	crlf := strings.ReplaceAll(string(lf), "\n", "\r\n")
	if err := os.WriteFile(path, []byte(crlf), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(dir, true); err != nil {
		t.Fatalf("-check called a CRLF checkout stale: %v", err)
	}
}

func TestRefusalsThatWouldOtherwiseProduceAWrongIndex(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, dir string)
		want  string
	}{
		{
			name: "duplicate number",
			setup: func(t *testing.T, dir string) {
				writeRecord(t, dir, "0002-a-second-file.md", good("0002", "A second file", "accepted", ""))
			},
			want: "number 0002 is used by both",
		},
		{
			name: "superseded without a pointer",
			setup: func(t *testing.T, dir string) {
				writeRecord(t, dir, "0004-replaced.md", good("0004", "Replaced", "superseded", ""))
			},
			want: "Superseded-By is missing",
		},
		{
			name: "status outside the permitted set",
			setup: func(t *testing.T, dir string) {
				writeRecord(t, dir, "0004-odd-status.md", good("0004", "Odd status", "draft", ""))
			},
			want: "is not one of proposed, accepted, superseded",
		},
		{
			name: "missing header field",
			setup: func(t *testing.T, dir string) {
				writeRecord(t, dir, "0004-no-date.md",
					"# 0004. No date\n\nNumber: 0004\nTitle: No date\nStatus: accepted\n\n## What was decided\n\nx\n")
			},
			want: "header field Date is missing",
		},
		{
			name: "filename number and heading disagree",
			setup: func(t *testing.T, dir string) {
				writeRecord(t, dir, "0004-mismatched.md", good("0006", "Mismatched", "accepted", ""))
			},
			want: "heading says 0006, filename says 0004",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := twoRecords(t)
			c.setup(t, dir)
			err := run(dir, false)
			if err == nil {
				t.Fatal("the generator wrote an index it should have refused")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %q does not mention %q", err, c.want)
			}
			if _, statErr := os.Stat(filepath.Join(dir, "README.md")); statErr == nil {
				t.Fatal("an index was written despite the refusal")
			}
		})
	}
}

func TestSupersededRecordPointsForwardInTheIndex(t *testing.T) {
	dir := twoRecords(t)
	writeRecord(t, dir, "0004-replaced.md", good("0004", "Replaced", "superseded", "Superseded-By: 0006\n"))
	writeRecord(t, dir, "0006-the-replacement.md", good("0006", "The replacement", "accepted", ""))
	if err := run(dir, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "| superseded by 0006 |") {
		t.Fatalf("the index does not carry the forward pointer:\n%s", got)
	}
}

// The committed index is a golden: a file this repository holds and reviews,
// produced by the command beside it. This is the assertion that keeps the two
// together, and it is the one a change to a record has to answer.
//
// It reads the real directory rather than a fixture, because a fixture would
// prove the renderer and not the file anybody reads. The comparison is the
// helper in internal/golden, so a stale index is reported as the record that
// differs rather than as a byte that stopped matching, and running this
// package's suite with -update rewrites it:
//
//	go test ./tools/decisionindex -update
func TestTheCommittedIndexIsWhatTheRecordsRenderTo(t *testing.T) {
	dir := filepath.Join("..", "..", "docs", "decisions")

	records, err := readRecords(dir)
	if err != nil {
		t.Fatal(err)
	}

	golden.Assert(t, filepath.Join(dir, "README.md"), render(records))
}
