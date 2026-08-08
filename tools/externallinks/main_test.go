package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tree writes documents into a directory of their own, so a case below says
// something about the reading rather than about this repository.
func tree(t *testing.T, files map[string]string) string {
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

func TestCollectReadsBothWaysADocumentWritesAUrlAndSaysWhere(t *testing.T) {
	root := tree(t, map[string]string{
		"README.md": "# T\n\nSee [the code](https://example.invalid/a).\n\n" +
			"The terms are at https://example.invalid/b, read once.\n",
		"docs/note.md": "# T\n\n[ref]: https://example.invalid/a\n",
		"main.go":      "package main\n",
	})

	found, err := collect(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 2 {
		t.Fatalf("read %d url(s), wanted 2: %v", len(found), found)
	}
	where := strings.Join(found["https://example.invalid/a"], " ")
	for _, want := range []string{"README.md:3", "docs/note.md:3"} {
		if !strings.Contains(where, want) {
			t.Errorf("the url written in two documents does not record %q: %s", want, where)
		}
	}
	if _, ok := found["https://example.invalid/b"]; !ok {
		t.Errorf("a url written as prose was not read: %v", found)
	}
}

// A URL inside a fence is quoted rather than written. Reading it would turn
// every example in a document into something somebody has to keep alive.
func TestCollectLeavesAUrlInsideAFenceAlone(t *testing.T) {
	root := tree(t, map[string]string{
		"README.md": "# T\n\n```\ncurl https://example.invalid/a\n```\n",
	})
	found, err := collect(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("read %v from inside a fence", found)
	}
}

func TestUrlsInStripsTheSentenceAroundThemAndReadsEachOnce(t *testing.T) {
	got := urlsIn("Both https://example.invalid/a. and https://example.invalid/a again, plus https://example.invalid/b;")
	want := []string{"https://example.invalid/a", "https://example.invalid/b"}
	if len(got) != len(want) {
		t.Fatalf("read %v, wanted %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("read %q, wanted %q", got[i], want[i])
		}
	}
}

// answered builds a fetcher that says whatever the test decided, so the reading
// is exercised without a socket. The tier this suite belongs to may not open
// one, which is the whole reason the asking is a parameter.
func answered(by map[string]answer) func(string) answer {
	return func(url string) answer { return by[url] }
}

func TestReportPassesWhereEveryLinkAnswers(t *testing.T) {
	var out strings.Builder
	err := report(&out, map[string][]string{"https://example.invalid/a": {"README.md:3"}},
		answered(map[string]answer{"https://example.invalid/a": {status: 200}}))

	if err != nil {
		t.Fatalf("a link that answered 200 was reported as dead: %v", err)
	}
	if !strings.Contains(out.String(), "1 external link(s) read, 0 of them did not answer") {
		t.Errorf("the run does not say what it read:\n%s", out.String())
	}
}

func TestReportNamesTheLinkTheStatusAndWhereItIsWritten(t *testing.T) {
	var out strings.Builder
	err := report(&out, map[string][]string{
		"https://example.invalid/a": {"README.md:3"},
		"https://example.invalid/b": {"docs/note.md:9", "docs/note.md:14"},
	}, answered(map[string]answer{
		"https://example.invalid/a": {status: 200},
		"https://example.invalid/b": {status: 404},
	}))

	if err == nil {
		t.Fatal("a link answering 404 was reported as alive")
	}
	for _, want := range []string{"https://example.invalid/b", "answered 404", "docs/note.md:9, docs/note.md:14", "2 external link(s) read, 1 of them did not answer"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the report does not say %q:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "https://example.invalid/a\n    answered") {
		t.Errorf("a link that answered was listed among the ones that did not:\n%s", out.String())
	}
}

// A host that cannot be reached at all is a different statement from one that
// answered with a status, and the report keeps them apart because only one of
// them is a claim about the page.
func TestReportKeepsAHostThatSaidNothingApartFromOneThatAnswered(t *testing.T) {
	var out strings.Builder
	err := report(&out, map[string][]string{"https://example.invalid/a": {"README.md:3"}},
		answered(map[string]answer{"https://example.invalid/a": {err: errors.New("no such host")}}))

	if err == nil {
		t.Fatal("a host that said nothing was reported as alive")
	}
	if !strings.Contains(out.String(), "no such host") {
		t.Errorf("the report does not carry what went wrong:\n%s", out.String())
	}
}
