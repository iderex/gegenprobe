package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// docTree writes documents into a tree of their own. Every fixture below is the
// passing document with one thing changed, so a verdict that moves between two
// of them moved for that thing and not for anything around it.
func docTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// good is the passing fixture for the form leg. Every case that expects a
// refusal below is this document with one character or one line different.
const good = "# A title\n" +
	"\n" +
	"A sentence about the tree.\n" +
	"\n" +
	"## A section\n" +
	"\n" +
	"Another sentence.\n"

func oneDoc(t *testing.T, body string) string {
	t.Helper()
	return docTree(t, map[string]string{"note.md": body})
}

func TestDocumentationFormAcceptsTheForm(t *testing.T) {
	if o := documentationFormLeg(oneDoc(t, good)); o.verdict != passed {
		t.Fatalf("a document in the form was refused: %s", o.detail)
	}
}

// The one character mistake somebody actually makes. A trailing space is
// invisible in the editor that made it, so this fixture differs from the passing
// one by a character nobody can see.
func TestDocumentationFormRefusesTrailingWhitespaceAndNamesTheLine(t *testing.T) {
	o := documentationFormLeg(oneDoc(t, strings.Replace(good, "A sentence about the tree.", "A sentence about the tree. ", 1)))

	if o.verdict != failed {
		t.Fatalf("a line ending in whitespace passed: %v %s", o.verdict, o.detail)
	}
	for _, want := range []string{"note.md:3", "ends in whitespace"} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the refusal does not say %q:\n%s", want, o.detail)
		}
	}
}

func TestDocumentationFormRefusesTheOtherWaysOfLeavingTheForm(t *testing.T) {
	for _, c := range []struct {
		name string
		body string
		want string
	}{
		{"a tab", strings.Replace(good, "Another sentence.", "Another\tsentence.", 1), "holds a tab"},
		{"two blank lines", strings.Replace(good, "\n\n## A section", "\n\n\n## A section", 1), "second blank line in a row"},
		{"a heading with no space", strings.Replace(good, "## A section", "##A section", 1), "one to six hashes"},
		{"a heading closed with hashes", strings.Replace(good, "## A section", "## A section ##", 1), "closed with hashes"},
		{"no newline at the end", strings.TrimSuffix(good, "\n"), "no newline after it"},
		{"a blank line at the end", good + "\n", "ends with a blank line"},
		{"a blank first line", "\n" + good, "starts with a blank line"},
		{"an unclosed fence", good + "\n```\ngo run ./gate\n", "never closed"},
		{"nothing at all", "", "the file is empty"},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := documentationFormLeg(oneDoc(t, c.body))
			if o.verdict != failed {
				t.Fatalf("%s passed: %v %s", c.name, o.verdict, o.detail)
			}
			if !strings.Contains(o.detail, c.want) {
				t.Errorf("the refusal does not say %q:\n%s", c.want, o.detail)
			}
		})
	}
}

// A fence carries bytes quoted from somewhere else. Every rule the form imposes
// is broken inside this one deliberately, and the leg has to leave all of it
// alone, or a document could not quote the output of a command that prints a
// tab.
func TestDocumentationFormLeavesAFencedBlockAlone(t *testing.T) {
	body := good + "\n```\ntrailing space here \ta tab\n\n\ntwo blank lines above\n###no space\n```\n"
	if o := documentationFormLeg(oneDoc(t, body)); o.verdict != passed {
		t.Fatalf("the leg judged the inside of a fence: %s", o.detail)
	}
}

func TestDocumentationFormJudgesAfterNormalisingCarriageReturnsAndSaysSo(t *testing.T) {
	o := documentationFormLeg(oneDoc(t, strings.ReplaceAll(good, "\n", "\r\n")))
	if o.verdict != passed {
		t.Fatalf("a document with CRLF endings was refused: %s", o.detail)
	}
	if !strings.Contains(o.detail, "CRLF") {
		t.Errorf("the pass does not say what it normalised:\n%s", o.detail)
	}
}

func TestTheThreeDocumentationLegsSkipWhereThereIsNoDocument(t *testing.T) {
	root := docTree(t, map[string]string{"main.go": "package main\n"})
	for name, leg := range map[string]func(string) outcome{
		"form":  documentationFormLeg,
		"links": documentationLinksLeg,
		"paths": documentationPathsLeg,
	} {
		if o := leg(root); o.verdict != skipped {
			t.Errorf("the %s leg reported %v over a tree with no document, which reads as a pass over something", name, o.verdict)
		}
	}
}

// linked is the passing fixture for the links leg: one link, to a file that is
// there.
const linked = "# A title\n\nSee [the form](docs/markdown-form.md).\n"

func TestDocumentationLinksResolvesALinkIntoTheTree(t *testing.T) {
	root := docTree(t, map[string]string{
		"note.md":                 linked,
		"docs/markdown-form.md":   good,
		"docs/decisions/index.md": good,
	})
	if o := documentationLinksLeg(root); o.verdict != passed {
		t.Fatalf("a link to a file that is there was refused: %s", o.detail)
	}
}

// The same document against a tree missing the one file it points at.
func TestDocumentationLinksRefusesALinkToNothingAndNamesIt(t *testing.T) {
	o := documentationLinksLeg(docTree(t, map[string]string{"note.md": linked}))

	if o.verdict != failed {
		t.Fatalf("a link to nothing passed: %v %s", o.verdict, o.detail)
	}
	for _, want := range []string{"note.md:3", "docs/markdown-form.md", "resolves to nothing"} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the refusal does not say %q:\n%s", want, o.detail)
		}
	}
}

func TestDocumentationLinksReadsWhatItSaysItReads(t *testing.T) {
	for _, c := range []struct {
		name string
		body string
		want verdict
	}{
		{"an external link", "# T\n\n[a code](https://example.invalid/x).\n", skipped},
		{"a bare fragment", "# T\n\n[a section](#a-section).\n", skipped},
		{"a link inside a fence", "# T\n\n```\n[gone](docs/gone.md)\n```\n", skipped},
		{"a link inside a code span", "# T\n\nWrite it as `[gone](docs/gone.md)` instead.\n", skipped},
		{"a fragment on a path that is there", "# T\n\n[here](note.md#a-title).\n", passed},
		{"a fragment on a path that is not", "# T\n\n[gone](docs/gone.md#a-title).\n", failed},
		{"a target read from the root", "# T\n\n[here](/note.md).\n", passed},
		{"a reference definition", "# T\n\nSee [it][ref].\n\n[ref]: docs/gone.md\n", failed},
		{"a target in angle brackets", "# T\n\n[here](<note.md>).\n", passed},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := documentationLinksLeg(oneDoc(t, c.body))
			if o.verdict != c.want {
				t.Fatalf("wanted %v, got %v: %s", c.want, o.verdict, o.detail)
			}
		})
	}
}

// named is the passing fixture for the paths leg: a path in a plain sentence,
// pointing at a directory that is there.
const named = "# A title\n\nThe reader lives in internal/fixture and nothing else reads one.\n"

func TestDocumentedPathsResolvesAPathNamedInProse(t *testing.T) {
	root := docTree(t, map[string]string{
		"note.md":                     named,
		"internal/fixture/fixture.go": "package fixture\n",
	})
	o := documentedPathsOutcome(t, root)
	if o.verdict != passed {
		t.Fatalf("a path that is there was refused: %s", o.detail)
	}
	if !strings.Contains(o.detail, "1 path(s)") {
		t.Errorf("the pass does not say how many paths it resolved:\n%s", o.detail)
	}
}

// The same sentence against a tree where the directory under internal is spelt
// one character differently, which is the mistake a rename leaves behind.
func TestDocumentedPathsRefusesAPathThatIsNotThereAndNamesIt(t *testing.T) {
	root := docTree(t, map[string]string{
		"note.md":                      named,
		"internal/fixtures/fixture.go": "package fixture\n",
	})
	o := documentedPathsOutcome(t, root)

	if o.verdict != failed {
		t.Fatalf("a path that is not there passed: %v %s", o.verdict, o.detail)
	}
	for _, want := range []string{"note.md:3", "internal/fixture", "not in this tree"} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the refusal does not say %q:\n%s", want, o.detail)
		}
	}
}

// The three shapes a document about this project has to write and which are not
// files in it. All three hold a slash and none has a first segment at the root,
// which is the whole of what separates them from a relative path.
func TestDocumentedPathsLeavesAloneWhatIsNotAPathHere(t *testing.T) {
	for _, c := range []struct {
		name     string
		sentence string
	}{
		{"a standard library import", "The tier refuses any import of os/exec, including net/http."},
		{"a platform pair", "The cross build covers linux/amd64 and darwin/arm64."},
		{"a bundle's own layout", "The manifest sits under bundle/ beside raw/."},
		{"a pinned module version", "Install honnef.co/go/tools/cmd/staticcheck@2025.1.1 first."},
		{"a package pattern", "Run go test ./... before pushing."},
		{"a URL", "The source is at https://example.invalid/a/b."},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := docTree(t, map[string]string{"note.md": "# A title\n\n" + c.sentence + "\n"})
			if o := documentationPathsLeg(root); o.verdict != skipped {
				t.Fatalf("%s was read as a path into this tree: %v %s", c.name, o.verdict, o.detail)
			}
		})
	}
}

func TestDocumentedPathsReadsABacktickSpanAndATrailingFullStop(t *testing.T) {
	for _, c := range []struct {
		name     string
		sentence string
	}{
		{"a backtick span", "The convention is `docs/gone.md` and nothing else."},
		{"a full stop after the path", "The convention is docs/gone.md."},
		{"a path in brackets", "The convention (docs/gone.md) says so."},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := docTree(t, map[string]string{
				"note.md":       "# A title\n\n" + c.sentence + "\n",
				"docs/other.md": good,
			})
			o := documentationPathsLeg(root)
			if o.verdict != failed {
				t.Fatalf("%s was not read: %v %s", c.name, o.verdict, o.detail)
			}
			if !strings.Contains(o.detail, "docs/gone.md") {
				t.Errorf("the refusal does not name the path:\n%s", o.detail)
			}
		})
	}
}

// A path a document names twice on one line is one path, so a count a reader
// takes for how many things were checked is not inflated by a repetition.
func TestDocumentedPathsCountsAPathNamedTwiceOnALineOnce(t *testing.T) {
	root := docTree(t, map[string]string{
		"note.md":       "# A title\n\nBoth docs/other.md and docs/other.md again.\n",
		"docs/other.md": good,
	})
	o := documentationPathsLeg(root)
	if o.verdict != passed {
		t.Fatalf("a path that is there was refused: %s", o.detail)
	}
	if !strings.Contains(o.detail, "1 path(s)") {
		t.Errorf("the same path on one line was counted twice:\n%s", o.detail)
	}
}

// documentedPathsOutcome runs the leg and fails the test where the walk itself
// broke, so a case below reads as a statement about the document rather than
// about the tree it was written into.
func documentedPathsOutcome(t *testing.T, root string) outcome {
	t.Helper()
	o := documentationPathsLeg(root)
	if o.verdict == skipped {
		t.Fatalf("the leg read no path at all: %s", o.detail)
	}
	return o
}

func TestMarkdownFilesReadsTheRootAndSkipsTheGitDirectory(t *testing.T) {
	root := docTree(t, map[string]string{
		"README.md":      good,
		"docs/note.md":   good,
		".git/COMMIT.md": good,
		"main.go":        "package main\n",
	})
	got, err := markdownFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"README.md": true, "docs/note.md": true}
	if len(got) != len(want) {
		t.Fatalf("read %v, wanted exactly %v", got, want)
	}
	for _, rel := range got {
		if !want[rel] {
			t.Errorf("read %q, which is not a document this leg's scope names", rel)
		}
	}
}
