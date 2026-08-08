package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fixtures below are one tree with one change made to it. The tree is three
// records and the index over them, and it holds a supersession in both
// directions so that the record pointing forward is a passing record rather than
// something only the failing fixture has.
//
// neighbour is the record no fixture touches. Every fixture asserts that the
// refusal does not name it, because a check that reddens the whole directory
// when one file is wrong sends a reader to the wrong file.
const neighbour = "0000-a-good-record.md"

// record renders a well formed record. Everything a fixture changes is a
// parameter, so the difference between a passing tree and a failing one is
// visible in the call rather than buried in a string.
func record(number, title, status, extraHeader, sections string) string {
	if sections == "" {
		sections = "## What was decided\n\nThis.\n\n## Why\n\nBecause.\n\n## What was rejected\n\nThat.\n\n## What this costs\n\nSomething.\n"
	}
	return "# " + number + ". " + title + "\n\n" +
		"Number: " + number + "\n" +
		"Title: " + title + "\n" +
		"Status: " + status + "\n" +
		"Date: 2026-08-07\n" +
		extraHeader +
		"\n" + sections
}

// index renders the README over exactly the files named, in the shape the
// generator writes, so the link extraction under test is reading a real index
// rather than a convenient one.
func index(files ...string) string {
	var b strings.Builder
	b.WriteString("# Decision records\n\n| Number | Title | Status | Date |\n| --- | --- | --- | --- |\n")
	for _, f := range files {
		b.WriteString("| [" + f[:4] + "](" + f + ") | Title | accepted | 2026-08-07 |\n")
	}
	return b.String()
}

// goodTree is the passing tree every fixture is one change away from.
func goodTree() map[string]string {
	files := map[string]string{
		neighbour:                   record("0000", "A good record", "accepted", "", ""),
		"0001-a-replaced-record.md": record("0001", "A replaced record", "superseded", "Superseded-By: 0002\n", ""),
		"0002-the-replacement.md":   record("0002", "The replacement", "accepted", "", ""),
	}
	files["README.md"] = index(neighbour, "0001-a-replaced-record.md", "0002-the-replacement.md")
	return files
}

// writeTree lays the files out under root/docs/decisions and returns root, which
// is what the leg is given.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "docs", "decisions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestTheGoodTreePasses(t *testing.T) {
	o := decisionRecordsLeg(writeTree(t, goodTree()))
	if o.verdict != passed {
		t.Fatalf("the tree every fixture is derived from does not pass: %s\n%s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "3 record(s) read") {
		t.Errorf("a passing run says %q, and it should say how many records it read", o.detail)
	}
}

// fixtures are the five refusals the format owes, each one change from goodTree.
// The index refusal is written in both directions, because the direction that
// goes stale in silence is a record added without a line rather than a line left
// pointing at nothing.
var fixtures = []struct {
	name   string
	change func(files map[string]string)
	// want is a phrase from the refusal, and names is the file the refusal has
	// to name so a reader knows where to go.
	want  string
	names string
}{
	{
		name: "one removed header field",
		change: func(files map[string]string) {
			files["0002-the-replacement.md"] = strings.ReplaceAll(files["0002-the-replacement.md"], "Date: 2026-08-07\n", "")
		},
		want:  "header field Date is missing",
		names: "0002-the-replacement.md",
	},
	{
		name: "one reordered section",
		change: func(files map[string]string) {
			files["0002-the-replacement.md"] = strings.ReplaceAll(
				files["0002-the-replacement.md"],
				"## Why\n\nBecause.\n\n## What was rejected\n\nThat.\n",
				"## What was rejected\n\nThat.\n\n## Why\n\nBecause.\n")
		},
		want:  "has its sections in the order",
		names: "0002-the-replacement.md",
	},
	{
		name: "one missing forward pointer",
		change: func(files map[string]string) {
			files["0001-a-replaced-record.md"] = strings.ReplaceAll(files["0001-a-replaced-record.md"], "Superseded-By: 0002\n", "")
		},
		want:  "status is superseded and Superseded-By names no record",
		names: "0001-a-replaced-record.md",
	},
	{
		name: "one duplicated number",
		change: func(files map[string]string) {
			files["0002-a-second-file.md"] = record("0002", "A second file", "accepted", "", "")
			files["README.md"] = index(neighbour, "0001-a-replaced-record.md", "0002-the-replacement.md", "0002-a-second-file.md")
		},
		want:  "number 0002 is used by",
		names: "0002-a-second-file.md",
	},
	{
		name: "one index line removed",
		change: func(files map[string]string) {
			files["README.md"] = index(neighbour, "0001-a-replaced-record.md")
		},
		want:  "does not list 0002-the-replacement.md",
		names: "README.md",
	},
	{
		name: "one index line for a record that is not there",
		change: func(files map[string]string) {
			files["README.md"] = index(neighbour, "0001-a-replaced-record.md", "0002-the-replacement.md", "0009-never-written.md")
		},
		want:  "lists 0009-never-written.md, and no such record is in this directory",
		names: "README.md",
	},
}

func TestEachFixtureIsRefusedAndTheRefusalSaysWhichFile(t *testing.T) {
	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			files := goodTree()
			f.change(files)
			o := decisionRecordsLeg(writeTree(t, files))

			if o.verdict != failed {
				t.Fatalf("the leg reported %s for %s, want a refusal\n%s", o.verdict, f.name, o.detail)
			}
			if !strings.Contains(o.detail, f.want) {
				t.Errorf("refusal does not say %q:\n%s", f.want, o.detail)
			}
			if !strings.Contains(o.detail, f.names) {
				t.Errorf("refusal does not name %s:\n%s", f.names, o.detail)
			}
			if f.names != neighbour && strings.Contains(o.detail, neighbour) {
				t.Errorf("refusal names %s, which is correct in every way and was not changed:\n%s", neighbour, o.detail)
			}
		})
	}
}

// legsThatJudgeADirectory is the part of the gate that can be pointed at a tree
// which is not a Go module. The legs left out shell out to the toolchain and
// would fail on any such tree for a reason that has nothing to do with these
// fixtures. That is the bound on the assertion below: it shows the decision
// records leg is what refuses each fixture among the legs that examine it, not
// among every leg the command has.
func legsThatJudgeADirectory(withDecisionRecords bool) []leg {
	ls := []leg{
		{name: "format", subject: "fixture", run: formatLeg},
		{name: "action pinning", subject: "fixture", run: actionPinningLeg},
	}
	if withDecisionRecords {
		ls = append(ls, leg{name: "decision records", subject: "fixture", run: decisionRecordsLeg})
	}
	return ls
}

// Disabling the leg has to turn every fixture green. A fixture that is refused
// with the leg removed is a fixture proving something else.
func TestDisablingTheLegTurnsEveryFixtureGreen(t *testing.T) {
	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			files := goodTree()
			f.change(files)
			root := writeTree(t, files)

			if status := run(io.Discard, root, legsThatJudgeADirectory(true)); status == 0 {
				t.Fatalf("the gate passed %s with the leg enabled", f.name)
			}
			if status := run(io.Discard, root, legsThatJudgeADirectory(false)); status != 0 {
				t.Errorf("the gate still refused %s with the leg disabled, so the fixture proves something else", f.name)
			}
		})
	}
}

// The refusals below are part of the format and are not among the five the
// issue names, so they are asserted here rather than in the fixture table.
func TestTheRestOfTheFormat(t *testing.T) {
	cases := []struct {
		name   string
		change func(files map[string]string)
		want   string
	}{
		{
			name: "a status outside the permitted set",
			change: func(files map[string]string) {
				files["0002-the-replacement.md"] = strings.ReplaceAll(files["0002-the-replacement.md"], "Status: accepted", "Status: draft")
			},
			want: `Status is "draft"`,
		},
		{
			name: "an empty section",
			change: func(files map[string]string) {
				files["0002-the-replacement.md"] = strings.ReplaceAll(files["0002-the-replacement.md"], "## What this costs\n\nSomething.\n", "## What this costs\n\n")
			},
			want: `section "What this costs" is empty`,
		},
		{
			name: "a section the format does not define",
			change: func(files map[string]string) {
				files["0002-the-replacement.md"] = strings.ReplaceAll(files["0002-the-replacement.md"], "## Why\n", "## Why\n\nBecause.\n\n## Notes\n")
			},
			want: `carries a level two heading "Notes"`,
		},
		{
			name: "a date that is not a date",
			change: func(files map[string]string) {
				files["0002-the-replacement.md"] = strings.ReplaceAll(files["0002-the-replacement.md"], "Date: 2026-08-07", "Date: 7 August 2026")
			},
			want: "which is not a YYYY-MM-DD date",
		},
		{
			name: "a forward pointer at a record that does not exist",
			change: func(files map[string]string) {
				files["0001-a-replaced-record.md"] = strings.ReplaceAll(files["0001-a-replaced-record.md"], "Superseded-By: 0002", "Superseded-By: 0009")
			},
			want: "is superseded by 0009, and no record with that number is in this directory",
		},
		{
			name: "a forward pointer on a record that is in force",
			change: func(files map[string]string) {
				files["0002-the-replacement.md"] = strings.ReplaceAll(files["0002-the-replacement.md"], "Date: 2026-08-07\n", "Date: 2026-08-07\nSuperseded-By: 0000\n")
			},
			want: "and the format carries that field only on a superseded record",
		},
		{
			name: "a heading whose number is not the filename's",
			change: func(files map[string]string) {
				files["0002-the-replacement.md"] = strings.ReplaceAll(files["0002-the-replacement.md"], "# 0002. The replacement", "# 0007. The replacement")
			},
			want: "heading says 0007 and the filename says 0002",
		},
		{
			name: "a record whose filename is not NNNN-short-slug.md",
			change: func(files map[string]string) {
				files["notes.md"] = record("0004", "Notes", "accepted", "", "")
			},
			want: "is not NNNN-short-slug.md",
		},
		{
			name: "no index at all",
			change: func(files map[string]string) {
				delete(files, "README.md")
			},
			want: "the index is missing",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files := goodTree()
			c.change(files)
			o := decisionRecordsLeg(writeTree(t, files))
			if o.verdict != failed {
				t.Fatalf("the leg reported %s, want a refusal\n%s", o.verdict, o.detail)
			}
			if !strings.Contains(o.detail, c.want) {
				t.Errorf("refusal does not say %q:\n%s", c.want, o.detail)
			}
		})
	}
}

// A tree with no decisions directory reports a skip. A pass there would be a
// green line standing for an examination that did not happen, which is the one
// thing this command exists to refuse.
func TestNoDecisionsDirectoryIsASkipRatherThanAPass(t *testing.T) {
	o := decisionRecordsLeg(t.TempDir())
	if o.verdict != skipped {
		t.Fatalf("verdict = %s for a tree with no docs/decisions, want a skip", o.verdict)
	}
	if !strings.Contains(o.detail, "docs/decisions") {
		t.Errorf("the skip says %q, and it should name what was missing", o.detail)
	}
}
