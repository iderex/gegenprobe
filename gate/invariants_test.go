package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Every rule is proven by a pair of trees that differ on one line. The passing
// member is refused by nothing, the other is refused by that rule and by no
// other, and a test below asserts the one line difference, so a fixture that
// could not have passed proves less than one that nearly did.
type invariantFixture struct {
	pass map[string]string
	trip map[string]string
	line int
	says string
}

const (
	aPackage = "package a\n"
	bPackage = "package b\n"
)

// invariantFixtures pairs every rule with the near miss somebody writes. The
// two counting rules are proven by a second package doing what one package may,
// which is the shape of the property rather than of a single file.
func invariantFixtures() map[string]invariantFixture {
	return map[string]invariantFixture{
		ruleCaseParse: {
			pass: map[string]string{
				"internal/a/a.go": aPackage + "\nimport _ \"gopkg.in/yaml.v3\"\n",
				"internal/b/b.go": bPackage + "\nimport _ \"encoding/json\"\n",
			},
			trip: map[string]string{
				"internal/a/a.go": aPackage + "\nimport _ \"gopkg.in/yaml.v3\"\n",
				"internal/b/b.go": bPackage + "\nimport _ \"gopkg.in/yaml.v3\"\n",
			},
			line: 3,
			says: "decodes the case file's own format",
		},
		ruleConstants: {
			pass: map[string]string{
				"internal/a/a.go": aPackage + "\nconst factor = 8065.543937349211\n",
				"internal/b/b.go": bPackage + "\nconst tolerance = 0.5\n",
			},
			trip: map[string]string{
				"internal/a/a.go": aPackage + "\nconst factor = 8065.543937349211\n",
				"internal/b/b.go": bPackage + "\nconst factor = 8065.543937349211\n",
			},
			line: 3,
			says: "digit numeric constant",
		},
		ruleRenderer: {
			pass: map[string]string{
				"internal/report/report.go": "package report\n\nimport \"os\"\n\nfunc Render() { _ = os.Getpid() }\n",
			},
			trip: map[string]string{
				"internal/report/report.go": "package report\n\nimport \"os\"\n\nfunc Render() { _, _ = os.ReadFile(\"x\") }\n",
			},
			line: 5,
			says: "calls os.ReadFile in the package that renders the report",
		},
		ruleNoVerdict: {
			pass: map[string]string{
				"internal/a/a.go": aPackage + "\nfunc spread(vs []float64) float64 { return vs[0] }\n",
			},
			trip: map[string]string{
				"internal/a/a.go": aPackage + "\nfunc average(vs []float64) float64 { return vs[0] }\n",
			},
			line: 3,
			says: "whose name says it averages the codes",
		},
		ruleSorted: {
			pass: map[string]string{
				"internal/a/a.go": aPackage + "\nimport (\n\t\"fmt\"\n\t\"io\"\n)\n\nfunc Write(w io.Writer, rows map[string]string, order []string) {\n" +
					"\tfor _, k := range order {\n\t\tfmt.Fprintln(w, rows[k])\n\t}\n}\n",
			},
			trip: map[string]string{
				"internal/a/a.go": aPackage + "\nimport (\n\t\"fmt\"\n\t\"io\"\n)\n\nfunc Write(w io.Writer, rows map[string]string, order []string) {\n" +
					"\tfor k := range rows {\n\t\tfmt.Fprintln(w, rows[k])\n\t}\n}\n",
			},
			line: 10,
			says: "inside a range over a map",
		},
		ruleTierTag: {
			pass: map[string]string{
				"internal/a/a_integration_test.go": "//go:build integration\n\n" + aPackage,
			},
			trip: map[string]string{
				"internal/a/a_integration_test.go": "//go:build regression\n\n" + aPackage,
			},
			line: 1,
			says: "carries no //go:build integration constraint",
		},
	}
}

func judgeFixture(t *testing.T, files map[string]string, on map[string]bool) []breach {
	t.Helper()
	found, _, err := judgeSources(files, on)
	if err != nil {
		t.Fatalf("a fixture did not parse: %v", err)
	}
	return found
}

func rulesTripped(found []breach) []string {
	seen := map[string]bool{}
	var out []string
	for _, b := range found {
		if !seen[b.rule] {
			seen[b.rule] = true
			out = append(out, b.rule)
		}
	}
	sort.Strings(out)
	return out
}

func TestEveryRuleHasAFixtureThatTripsItAndNothingElse(t *testing.T) {
	for rule, f := range invariantFixtures() {
		t.Run(rule, func(t *testing.T) {
			if found := judgeFixture(t, f.pass, enabled(invariantIDs())); len(found) != 0 {
				t.Fatalf("the passing fixture for %s was refused: %v", rule, found)
			}

			found := judgeFixture(t, f.trip, enabled(invariantIDs()))
			if len(found) == 0 {
				t.Fatalf("the fixture for %s was not refused", rule)
			}
			if tripped := rulesTripped(found); len(tripped) != 1 || tripped[0] != rule {
				t.Fatalf("the fixture for %s tripped %v; a fixture tripping two rules proves neither", rule, tripped)
			}

			last := found[len(found)-1]
			if last.line != f.line {
				t.Errorf("the refusal names line %d; the fixture puts the site at line %d", last.line, f.line)
			}
			if !strings.Contains(last.why, f.says) {
				t.Errorf("the refusal does not say why in the terms the record uses: %q", last.why)
			}
		})
	}
}

// The near miss is one line. A fixture differing in two places proves the rule
// fired and not what it fired on.
func TestEachInvariantFixtureIsOneLineFromThePassingOne(t *testing.T) {
	for rule, f := range invariantFixtures() {
		t.Run(rule, func(t *testing.T) {
			good := strings.Split(flatten(f.pass), "\n")
			bad := strings.Split(flatten(f.trip), "\n")
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
				t.Fatalf("the fixture for %s differs on %d lines", rule, differing)
			}
		})
	}
}

// flatten renders a tree in path order so two trees can be compared line by
// line. The path goes in, because a fixture that moved a file rather than
// changing a line would otherwise read as one line different.
func flatten(files map[string]string) string {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var b strings.Builder
	for _, p := range paths {
		b.WriteString("== " + p + "\n")
		b.WriteString(files[p])
	}
	return b.String()
}

// Turning one rule off has to turn its own fixture green and leave the others
// where they were. A fixture green with its rule disabled is one the rule
// refused, rather than one nothing was ever asked about.
func TestDisablingARuleTurnsItsFixtureGreen(t *testing.T) {
	for rule, f := range invariantFixtures() {
		t.Run(rule, func(t *testing.T) {
			on := enabled(invariantIDs())
			delete(on, rule)

			if found := judgeFixture(t, f.trip, on); len(found) != 0 {
				t.Fatalf("with %s disabled its fixture is still refused: %v", rule, found)
			}
			if found := judgeFixture(t, f.trip, enabled(invariantIDs())); len(found) == 0 {
				t.Fatalf("with every rule enabled the same fixture is not refused")
			}
		})
	}
}

func TestWithEveryRuleDisabledEveryFixtureIsGreen(t *testing.T) {
	off := map[string]bool{}
	for rule, f := range invariantFixtures() {
		if found := judgeFixture(t, f.trip, off); len(found) != 0 {
			t.Errorf("the fixture for %s is refused by a lint that judges nothing: %v", rule, found)
		}
	}
}

// A rule with no fixture is a rule nothing proves bites, and it is refused here
// rather than shipped on the strength of being quotable.
func TestTheFixturesCoverTheRuleListExactly(t *testing.T) {
	f := invariantFixtures()
	seen := map[string]bool{}
	for _, id := range invariantIDs() {
		if _, ok := f[id]; !ok {
			t.Errorf("%s is a rule with no fixture that trips it, so nothing proves it bites", id)
		}
		if seen[id] {
			t.Errorf("%s is on the rule list twice", id)
		}
		seen[id] = true
	}
	for id := range f {
		if !seen[id] {
			t.Errorf("there is a fixture for %s, which is not on the rule list", id)
		}
	}
}

// Every rule sends whoever trips it to the record that argued it, and the leg
// refuses a rule naming a record the tree does not hold. The two trees below
// differ by exactly that record.
func TestARuleNamingARecordThatIsNotThereIsRefused(t *testing.T) {
	whole := invariantTree(t, nil)
	if o := invariantsLeg(whole); o.verdict == failed {
		t.Fatalf("a tree holding every named record was refused: %s", o.detail)
	}

	for _, inv := range invariantList() {
		t.Run(inv.id, func(t *testing.T) {
			root := invariantTree(t, map[string]bool{inv.record: true})

			o := invariantsLeg(root)
			if o.verdict != failed {
				t.Fatalf("%s names %s, which is not in this tree, and the leg passed", inv.id, inv.record)
			}
			for _, want := range []string{inv.id, inv.record} {
				if !strings.Contains(o.detail, want) {
					t.Errorf("the failure does not carry %q:\n%s", want, o.detail)
				}
			}
		})
	}
}

// invariantTree writes a module holding one Go file and the record every rule
// names, minus the ones the caller asked to leave out.
func invariantTree(t *testing.T, without map[string]bool) string {
	t.Helper()
	root := t.TempDir()

	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("go.mod", "module "+testModule+"\n\ngo 1.24\n")
	write("internal/a/a.go", aPackage)
	for _, inv := range invariantList() {
		if without[inv.record] {
			continue
		}
		write(inv.record, "# a record\n")
	}
	return root
}

func TestTheLegNamesTheFileTheLineAndTheRule(t *testing.T) {
	root := invariantTree(t, nil)
	f := invariantFixtures()[ruleNoVerdict]
	for rel, body := range f.trip {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	o := invariantsLeg(root)

	if o.verdict != failed {
		t.Fatalf("a tree declaring an averaging function passed the leg: %v %s", o.verdict, o.detail)
	}
	for _, want := range []string{"internal/a/a.go:3", ruleNoVerdict, "docs/decisions/0006-what-agreement-means.md"} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the failure does not carry %q:\n%s", want, o.detail)
		}
	}
}

// A rule that found nothing to judge is said out loud. A green line standing for
// no examination at all is the shape this whole command exists to refuse, and
// four of these rules have no subject in this tree until the packages they are
// about are written.
func TestThePassSaysWhichRulesFoundNothingToJudge(t *testing.T) {
	root := invariantTree(t, nil)

	o := invariantsLeg(root)

	if o.verdict != passed {
		t.Fatalf("a clean tree was not passed: %v %s", o.verdict, o.detail)
	}
	for _, want := range []string{ruleCaseParse, ruleConstants, ruleRenderer, ruleTierTag, "covers less than the rule list"} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the pass does not say that %q found nothing:\n%s", want, o.detail)
		}
	}
}

func TestTheLegSkipsWhereThereIsNoGoSourceToRead(t *testing.T) {
	root := invariantTree(t, nil)
	if err := os.Remove(filepath.Join(root, filepath.FromSlash("internal/a/a.go"))); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "go.mod")); err != nil {
		t.Fatal(err)
	}

	o := invariantsLeg(root)

	if o.verdict != skipped {
		t.Fatalf("a tree with no Go source did not report a skip: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "nothing was read") {
		t.Errorf("the skip does not say what was not examined: %q", o.detail)
	}
}

// One package doing a thing is the property holding, not the first half of a
// refusal. The counting rules refuse the second package and say how many there
// are, because which of them should have kept it is not this leg's judgement.
func TestOnePackageMayDoWhatTwoMayNot(t *testing.T) {
	for _, rule := range []string{ruleCaseParse, ruleConstants} {
		t.Run(rule, func(t *testing.T) {
			f := invariantFixtures()[rule]
			found := judgeFixture(t, f.trip, enabled(invariantIDs()))
			if len(found) != 2 {
				t.Fatalf("two packages doing it produced %d refusal(s); both are named", len(found))
			}
			if !strings.Contains(found[0].why, "2 packages in this tree do it") {
				t.Errorf("the refusal does not say how many packages do it: %q", found[0].why)
			}
		})
	}
}

// Source under a testdata directory is not compiled by the go tool, so it is not
// part of this module and a fixture written to be bad must not redden the tree
// that holds it.
func TestSourceUnderTestdataIsNotJudgedByTheLint(t *testing.T) {
	root := invariantTree(t, nil)
	dir := filepath.Join(root, filepath.FromSlash("internal/a/testdata"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.go"), []byte(aPackage+"\nfunc average() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if o := invariantsLeg(root); o.verdict != passed {
		t.Fatalf("source under testdata was judged: %v %s", o.verdict, o.detail)
	}
}

// A word is a word rather than a substring. Refusing meaning because it holds
// mean is the false red record 0009 answers with a narrower rule, and this is
// the narrowing.
func TestARuleAboutAWordIsNotARuleAboutASubstring(t *testing.T) {
	for _, name := range []string{"meaning", "Rankine", "demeanour", "averageless"} {
		src := map[string]string{"internal/a/a.go": aPackage + "\nfunc " + name + "() {}\n"}
		if found := judgeFixture(t, src, enabled(invariantIDs())); len(found) != 0 {
			t.Errorf("%s was refused, and it names no verdict: %v", name, found)
		}
	}
	for _, name := range []string{"averageOfTheCodes", "RankTheCodes", "with_mean_value", "winner"} {
		src := map[string]string{"internal/a/a.go": aPackage + "\nfunc " + name + "() {}\n"}
		if found := judgeFixture(t, src, enabled(invariantIDs())); len(found) != 1 {
			t.Errorf("%s names a verdict and was not refused", name)
		}
	}
}

// Collecting the keys and sorting them before the write is what the rule asks
// for, so a range over a slice built from a map is not a range over a map.
func TestSortedKeysAreNotAMapRange(t *testing.T) {
	src := map[string]string{
		"internal/a/a.go": aPackage + "\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"sort\"\n)\n\n" +
			"func Write(w io.Writer, rows map[string]string) {\n" +
			"\tvar keys []string\n" +
			"\tfor k := range rows {\n\t\tkeys = append(keys, k)\n\t}\n" +
			"\tsort.Strings(keys)\n" +
			"\tfor _, k := range keys {\n\t\tfmt.Fprintln(w, rows[k])\n\t}\n}\n",
	}
	if found := judgeFixture(t, src, enabled(invariantIDs())); len(found) != 0 {
		t.Errorf("a sorted write was refused: %v", found)
	}
}

// A test file naming no tier is a gate test and is left alone, which is the
// default record 0009 gives a contributor who decides nothing.
func TestATestFileNamingNoTierIsNotJudged(t *testing.T) {
	src := map[string]string{"internal/a/a_test.go": aPackage}
	if found := judgeFixture(t, src, enabled(invariantIDs())); len(found) != 0 {
		t.Errorf("an ordinary gate test was refused: %v", found)
	}
}

func TestTheLegSaysWhereASourceFileDoesNotParse(t *testing.T) {
	root := invariantTree(t, nil)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash("internal/a/a.go")), []byte("package\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if o := invariantsLeg(root); o.verdict != failed {
		t.Fatalf("a tree holding source that does not parse passed: %v %s", o.verdict, o.detail)
	}
}
