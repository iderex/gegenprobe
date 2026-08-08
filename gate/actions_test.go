package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// workflow writes one workflow file into a tree shaped like this repository's.
// The fixtures below differ from each other by one line and nothing else, so a
// verdict that moves between them moved for the reference and not for anything
// around it.
func workflow(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "check.yml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

const sha = "3d3c42e5aac5ba805825da76410c181273ba90b1"

// oneStep is the passing fixture. Every case below substitutes exactly the
// reference on the uses line.
func oneStep(ref string) string {
	return "name: Check\n" +
		"on:\n" +
		"  pull_request:\n" +
		"jobs:\n" +
		"  check:\n" +
		"    runs-on: ubuntu-latest\n" +
		"    steps:\n" +
		"      - name: Checkout Repository\n" +
		"        uses: " + ref + "\n" +
		"        with:\n" +
		"          persist-credentials: false\n"
}

func TestActionPinningAcceptsACommitSha(t *testing.T) {
	if o := actionPinningLeg(workflow(t, oneStep("actions/checkout@"+sha+" # v7.0.1"))); o.verdict != passed {
		t.Fatalf("a sha pinned reference was refused: %s", o.detail)
	}
}

// The one character mistake somebody actually makes: the tag left where the sha
// belongs. This fixture differs from the passing one by exactly the reference.
func TestActionPinningRefusesATagAndNamesTheLine(t *testing.T) {
	o := actionPinningLeg(workflow(t, oneStep("actions/checkout@v7.0.1")))

	if o.verdict != failed {
		t.Fatalf("an action pinned to a tag passed: %v %s", o.verdict, o.detail)
	}
	for _, want := range []string{"check.yml:9", "actions/checkout@v7.0.1", "not a 40 character commit sha"} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the failure does not carry %q:\n%s", want, o.detail)
		}
	}
}

func TestTheTwoFixturesDifferByExactlyTheReference(t *testing.T) {
	good := strings.Split(oneStep("actions/checkout@"+sha+" # v7.0.1"), "\n")
	bad := strings.Split(oneStep("actions/checkout@v7.0.1"), "\n")

	if len(good) != len(bad) {
		t.Fatalf("the fixtures differ in length, %d lines against %d", len(good), len(bad))
	}
	differing := 0
	for i := range good {
		if good[i] != bad[i] {
			differing++
		}
	}
	if differing != 1 {
		t.Fatalf("the fixtures differ on %d lines; the near miss proves less than one line apart would", differing)
	}
}

func TestActionPinningRefusesTheOtherWaysAReferenceIsNotPinned(t *testing.T) {
	for _, c := range []struct {
		name string
		ref  string
		want string
	}{
		{"a branch", "actions/checkout@main", "not a 40 character commit sha"},
		{"a short sha", "actions/checkout@3d3c42e", "not a 40 character commit sha"},
		{"no ref at all", "actions/checkout", "carries no ref at all"},
		{"an image by tag", "docker://alpine:3.20", "names a container image by tag"},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := actionPinningLeg(workflow(t, oneStep(c.ref)))
			if o.verdict != failed {
				t.Fatalf("%s passed: %s", c.ref, o.detail)
			}
			if !strings.Contains(o.detail, c.want) {
				t.Errorf("the failure does not say why:\n%s", o.detail)
			}
		})
	}
}

func TestActionPinningAcceptsWhatCannotBePinnedToASha(t *testing.T) {
	for _, c := range []struct{ name, ref string }{
		{"an action in this repository", "./.github/actions/build"},
		{"an image by digest", "docker://alpine@sha256:" + strings.Repeat("a", 64)},
	} {
		t.Run(c.name, func(t *testing.T) {
			if o := actionPinningLeg(workflow(t, oneStep(c.ref))); o.verdict != passed {
				t.Fatalf("%s was refused: %s", c.ref, o.detail)
			}
		})
	}
}

// A commented out reference is not a reference. Refusing one would push somebody
// to delete the comment rather than to pin anything.
func TestActionPinningIgnoresACommentedLine(t *testing.T) {
	body := oneStep("actions/checkout@"+sha+" # v7.0.1") + "        # uses: actions/setup-go@v6\n"
	if o := actionPinningLeg(workflow(t, body)); o.verdict != passed {
		t.Fatalf("a commented reference was refused: %s", o.detail)
	}
}

// Where there is nothing to read the leg says so rather than passing. A pass
// here would be a green line standing for no examination at all.
func TestActionPinningSkipsWhereThereIsNoWorkflowToRead(t *testing.T) {
	o := actionPinningLeg(t.TempDir())
	if o.verdict != skipped {
		t.Fatalf("a tree with no workflows did not report a skip: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "no action reference was read") {
		t.Errorf("the skip does not say what was not examined: %q", o.detail)
	}
}
