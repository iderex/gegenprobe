package commit

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iderex/gegenprobe/internal/fixture"
)

// declaration is what docs/commit-messages.md says, written here rather than
// read from there. A test reading that file would be asserting the file and the
// parser together, and the parser is what these tests are about; the leg named
// commit hygiene is what reads the real document, and it fails where the real
// document stops parsing.
const declaration = `
Exempt-Author: dependabot[bot]
Exempt-Author: github-actions[bot]

Allowed: U+000A
Allowed: U+0020-U+007E
`

func rules(t *testing.T) Rules {
	t.Helper()
	r, err := ParseRules(declaration)
	if err != nil {
		t.Fatalf("ParseRules: %v", err)
	}
	return r
}

// load reads one recorded range and parses it, so that every test below is run
// against bytes stored the way docs/fixtures.md fixes rather than against a
// literal a checkout is free to rewrite. The no-break space fixture is the
// reason that matters here: raw, it is one keystroke from the passing neighbour
// and nothing in a diff would show which file held which.
func load(t *testing.T, name string) []Commit {
	t.Helper()
	f, err := fixture.Load(filepath.Join("testdata", name+".fixture"))
	if err != nil {
		t.Fatalf("loading %s: %v", name, err)
	}
	commits, err := Parse(f.Bytes)
	if err != nil {
		t.Fatalf("parsing %s: %v", name, err)
	}
	if len(commits) == 0 {
		t.Fatalf("%s parsed to no commits", name)
	}
	return commits
}

func TestASubjectNamingItsIssuePasses(t *testing.T) {
	findings := Judge(load(t, "subject-names-the-issue"), rules(t), Internal)
	if len(findings) != 0 {
		t.Fatalf("the passing neighbour produced %d finding(s): %v", len(findings), findings)
	}
}

func TestASubjectNamingNoIssueIsRefused(t *testing.T) {
	findings := Judge(load(t, "subject-names-no-issue"), rules(t), Internal)
	if len(findings) != 1 {
		t.Fatalf("want exactly one finding, got %d: %v", len(findings), findings)
	}
	if findings[0].Rule != SubjectNamesNoIssue {
		t.Errorf("want the subject rule, got rule %d", findings[0].Rule)
	}
	if !findings[0].Fatal {
		t.Error("a subject naming no issue on this repository's own branch has to refuse the change")
	}
	if !Failed(findings) {
		t.Error("Failed says the change is not refused")
	}
}

// The two fixtures above differ in the issue reference and in nothing else, so
// the one that fails could have passed. That is the near miss the check is for.
func TestTheTwoSubjectFixturesDifferOnlyInTheReference(t *testing.T) {
	passing := load(t, "subject-names-the-issue")[0].Message
	failing := load(t, "subject-names-no-issue")[0].Message
	if strings.Replace(passing, " (#65)", "", 1) != failing {
		t.Fatalf("the fixtures differ in more than the reference:\n%q\n%q", passing, failing)
	}
}

func TestAMergeSubjectNamingNoIssueIsExempt(t *testing.T) {
	commits := load(t, "merge-names-no-issue")
	if commits[0].Parents != 2 {
		t.Fatalf("the fixture is not a merge: %d parent(s)", commits[0].Parents)
	}
	if findings := Judge(commits, rules(t), Internal); len(findings) != 0 {
		t.Fatalf("a merge produced %d finding(s): %v", len(findings), findings)
	}
}

func TestATrustedBotSubjectNamingNoIssueIsExempt(t *testing.T) {
	if findings := Judge(load(t, "bot-subject-names-no-issue"), rules(t), Internal); len(findings) != 0 {
		t.Fatalf("a trusted bot produced %d finding(s): %v", len(findings), findings)
	}
}

// An author whose name merely ends in the bot suffix is not exempt. The suffix
// is a convention on the forge rather than a fact about the account, so the
// declaration names accounts exactly and this is what holds it to that.
func TestAnUntrustedNameEndingInTheBotSuffixIsNotExempt(t *testing.T) {
	c := Commit{
		SHA:     "8888888888888888888888888888888888888888",
		Author:  "someone-else[bot] <nobody@example.invalid>",
		Parents: 1,
		Message: "Subject naming no issue\n",
	}
	if findings := Judge([]Commit{c}, rules(t), Internal); len(findings) != 1 {
		t.Fatalf("want one finding, got %d: %v", len(findings), findings)
	}
}

func TestACharacterOutsideTheAllowlistNamesTheCommitTheCodePointAndTheLine(t *testing.T) {
	commits := load(t, "no-break-space-in-the-subject")
	findings := Judge(commits, rules(t), Internal)
	if len(findings) != 1 {
		t.Fatalf("want exactly one finding, got %d: %v", len(findings), findings)
	}
	f := findings[0]
	if f.Rule != CharacterNotAllowed {
		t.Errorf("want the character rule, got rule %d", f.Rule)
	}
	if f.SHA != commits[0].SHA {
		t.Errorf("the finding names %q and not the commit %q", f.SHA, commits[0].SHA)
	}
	if f.Line != 1 {
		t.Errorf("the no-break space is on line 1, and the finding says line %d", f.Line)
	}
	if !strings.Contains(f.Detail, "U+00A0") {
		t.Errorf("the finding does not name the code point: %q", f.Detail)
	}
	if !strings.Contains(f.String(), commits[0].SHA) || !strings.Contains(f.String(), "line 1") {
		t.Errorf("the printed finding does not carry the commit and the line: %q", f.String())
	}
}

// The character that fails is invisible, which is why it is worth refusing and
// why the message names it by code point rather than printing it. This asserts
// the fixture really is one code point away from the passing neighbour.
func TestTheNoBreakSpaceFixtureIsOneCodePointFromThePassingNeighbour(t *testing.T) {
	passing := load(t, "subject-names-the-issue")[0].Message
	offending := load(t, "no-break-space-in-the-subject")[0].Message
	if strings.Replace(offending, "\u00a0", " ", 1) != passing {
		t.Fatalf("the fixtures differ in more than the one code point:\n%q\n%q", passing, offending)
	}
}

// Disabling the rule turns the offending fixture green, which is what says the
// fixture is proving the rule rather than something else about the bytes.
func TestDisablingTheAllowlistTurnsTheOffendingFixtureGreen(t *testing.T) {
	everything := Rules{Allowed: []Span{{Lo: 0, Hi: 0x10FFFF}}, Bots: rules(t).Bots}
	if findings := Judge(load(t, "no-break-space-in-the-subject"), everything, Internal); len(findings) != 0 {
		t.Fatalf("with every code point permitted the fixture still produced %d finding(s): %v", len(findings), findings)
	}
}

func TestASubjectFindingIsReportedAndNotRefusedForAContributionFromOutside(t *testing.T) {
	findings := Judge(load(t, "subject-names-no-issue"), rules(t), External)
	if len(findings) != 1 {
		t.Fatalf("want one finding, got %d: %v", len(findings), findings)
	}
	if findings[0].Fatal {
		t.Error("a contribution from outside cannot know an issue number, so this reports rather than refuses")
	}
	if Failed(findings) {
		t.Error("Failed refuses a change that only carries a reported finding")
	}
}

// The character rule is about what the message is made of rather than about what
// its author could have known, so it is not relaxed for anybody.
func TestTheCharacterRuleIsNotRelaxedForAContributionFromOutside(t *testing.T) {
	findings := Judge(load(t, "no-break-space-in-the-subject"), rules(t), External)
	if len(findings) != 1 || !findings[0].Fatal {
		t.Fatalf("want one refusing finding, got %v", findings)
	}
}

func TestReportSaysWhatItExaminedEvenWhenItFoundNothing(t *testing.T) {
	commits := load(t, "subject-names-the-issue")
	var b bytes.Buffer
	Report(&b, commits, nil, Internal)
	if !strings.Contains(b.String(), "1 commit(s) examined, 0 finding(s), 0 of them refusing.") {
		t.Fatalf("the report does not say what it covered: %q", b.String())
	}
}

func TestReportSaysWhichFindingsRefuseAndWhichAreOnlyReported(t *testing.T) {
	commits := load(t, "subject-names-no-issue")
	var b bytes.Buffer
	Report(&b, commits, Judge(commits, rules(t), External), External)
	out := b.String()
	if !strings.Contains(out, "reported: ") {
		t.Errorf("a reported finding is not marked as one: %q", out)
	}
	if strings.Contains(out, "refused: ") {
		t.Errorf("nothing here refuses, and the report says something does: %q", out)
	}
	if !strings.Contains(out, "not a branch of this repository") {
		t.Errorf("the report does not say why the finding did not refuse: %q", out)
	}
}

func TestParseRulesReadsTheSpansAndTheExemptAuthors(t *testing.T) {
	r := rules(t)
	if len(r.Allowed) != 2 {
		t.Fatalf("want two spans, got %d: %v", len(r.Allowed), r.Allowed)
	}
	for _, c := range []rune{'\n', ' ', 'A', '~'} {
		if !r.permits(c) {
			t.Errorf("U+%04X is inside the declaration and was refused", c)
		}
	}
	for _, c := range []rune{'\t', '\r', 0x00A0, 0x2010, 0x202E} {
		if r.permits(c) {
			t.Errorf("U+%04X is outside the declaration and was permitted", c)
		}
	}
	if len(r.Bots) != 2 {
		t.Fatalf("want two exempt authors, got %v", r.Bots)
	}
}

// A declaration that parses to nothing would refuse every commit in the tree,
// and a parser that answered an empty allowlist quietly is how that would first
// be met.
func TestADeclarationWithNoAllowedLineIsRefused(t *testing.T) {
	if _, err := ParseRules("# a document that declares nothing\n"); err == nil {
		t.Fatal("a declaration with no Allowed: line parsed")
	}
}

func TestARangeThatRunsBackwardsIsRefused(t *testing.T) {
	if _, err := ParseRules("Allowed: U+007E-U+0020\n"); err == nil {
		t.Fatal("a span running backwards parsed")
	}
}

func TestParseRefusesARecordWithTheWrongFieldCount(t *testing.T) {
	if _, err := Parse([]byte("1111\x1fauthor\x1fsubject")); err == nil {
		t.Fatal("a record with three fields parsed")
	}
}

func TestParseReadsAnEmptyStreamAsNoCommits(t *testing.T) {
	commits, err := Parse([]byte("\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(commits) != 0 {
		t.Fatalf("want no commits, got %d", len(commits))
	}
}

func TestSubjectIsTheWholeMessageWhereThereIsNoLineFeed(t *testing.T) {
	c := Commit{Message: "One line and no more (#65)"}
	if c.Subject() != c.Message {
		t.Fatalf("want %q, got %q", c.Message, c.Subject())
	}
}

func TestEveryRefusedCharacterIsDescribedWithoutBeingPrinted(t *testing.T) {
	for _, c := range []rune{'\t', '\r', 0x0007, 0x007F, 0x00A0, 0x2010} {
		d := describe(c)
		if d == "" {
			t.Errorf("U+%04X has no description", c)
		}
		if strings.ContainsRune(d, c) {
			t.Errorf("the description of U+%04X prints the character, which is what makes it invisible: %q", c, d)
		}
	}
}

// A line feed outside the allowlist has to be findable too, or a declaration
// that dropped it would refuse nothing while every message held one.
func TestALineFeedOutsideTheAllowlistIsFound(t *testing.T) {
	printableOnly := Rules{Allowed: []Span{{Lo: 0x20, Hi: 0x7E}}}
	c := Commit{SHA: "9999", Message: "Subject (#65)\nand a body\n"}
	findings := Judge([]Commit{c}, printableOnly, Internal)
	if len(findings) != 2 {
		t.Fatalf("want a finding per line feed, got %d: %v", len(findings), findings)
	}
	if findings[0].Line != 1 || findings[1].Line != 2 {
		t.Fatalf("the line numbers are wrong: %v", findings)
	}
}

func TestTheLineNumberFollowsTheMessage(t *testing.T) {
	c := Commit{SHA: "aaaa", Message: "Subject (#65)\n\nA body whose third line holds a\u00a0no-break space.\n"}
	findings := Judge([]Commit{c}, rules(t), Internal)
	if len(findings) != 1 {
		t.Fatalf("want one finding, got %d: %v", len(findings), findings)
	}
	if findings[0].Line != 3 {
		t.Fatalf("want line 3, got %d", findings[0].Line)
	}
}

func TestACodePointAboveTheHighestThereIsIsRefused(t *testing.T) {
	if _, err := ParseRules("Allowed: U+FFFFFF\n"); err == nil {
		t.Fatal("a number above the highest code point parsed as one")
	}
	if _, err := ParseRules("Allowed: U+0020-U+FFFFFF\n"); err == nil {
		t.Fatal("a span ending above the highest code point parsed")
	}
}

func TestAMalformedRecordIsNamedWithoutPastingTheWholeCommit(t *testing.T) {
	long := strings.Repeat("x", 200)
	_, err := Parse([]byte("1111\x1fauthor\x1f" + long))
	if err == nil {
		t.Fatal("a record with three fields parsed")
	}
	if len(err.Error()) > 150 {
		t.Errorf("the error pastes the whole record: %q", err)
	}
}

func TestAnASCIICharacterOutsideTheSpansIsStillDescribed(t *testing.T) {
	if describe('A') == "" {
		t.Fatal("an ASCII character outside the spans has no description")
	}
}

// Run reads the declaration before it reaches git, so both of these stop before
// a range is resolved and neither needs a repository.
func TestRunRefusesADeclarationItCannotRead(t *testing.T) {
	if _, err := Run(io.Discard, t.TempDir(), "not-here.md", "base", "head", Internal); err == nil {
		t.Fatal("a missing declaration was read")
	}
}

func TestRunRefusesADeclarationThatDeclaresNothing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "declaration.md"), []byte("# nothing here\n"), 0o644); err != nil {
		t.Fatalf("writing the declaration: %v", err)
	}
	_, err := Run(io.Discard, dir, "declaration.md", "base", "head", Internal)
	if err == nil {
		t.Fatal("a declaration holding no allowed span was accepted")
	}
	if !strings.Contains(err.Error(), "declaration.md") {
		t.Errorf("the error does not name the document it read: %v", err)
	}
}

func TestReportMarksARefusingFindingAsRefused(t *testing.T) {
	commits := load(t, "subject-names-no-issue")
	var b bytes.Buffer
	Report(&b, commits, Judge(commits, rules(t), Internal), Internal)
	out := b.String()
	if !strings.Contains(out, "refused: ") {
		t.Errorf("a refusing finding is not marked as one: %q", out)
	}
	if !strings.Contains(out, "1 commit(s) examined, 1 finding(s), 1 of them refusing.") {
		t.Errorf("the report does not count what it refused: %q", out)
	}
}
