package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// absenceTree writes one workflow file at the path the leg reads, inside a tree
// shaped like this repository's. Every case below differs from the passing
// fixture by one line and nothing else, so a verdict that moves between them
// moved for that line.
func absenceTree(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, filepath.FromSlash(absenceWorkflow))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// admissible is the passing fixture: the shape the real file has, reduced to
// the lines this leg reads.
func admissible() []string {
	return []string{
		"name: Gate tier with the capabilities absent",
		"on:",
		"  pull_request:",
		"jobs:",
		"  absent:",
		"    runs-on: ubuntu-latest",
		"    steps:",
		"      - name: Establish that no display is set or reachable",
		"        run: echo display",
		"      - name: Establish that no container engine is on the path",
		"        run: echo engine",
		"      - name: Establish that outbound network is refused",
		"        run: echo network",
		"      - name: Establish that no elevation tool is available",
		"        run: echo elevation",
		"      - name: Say what could not be established",
		"        run: cat not-removed.txt",
		"      - name: Run the gate tier",
		"        run: go test ./...",
	}
}

func joined(lines []string) string { return strings.Join(lines, "\n") + "\n" }

// without returns the fixture with the one line carrying the text removed. It
// fails the test rather than returning the fixture unchanged, because a case
// that silently deleted nothing would pass for the wrong reason.
func without(t *testing.T, text string) string {
	t.Helper()
	var out []string
	removed := 0
	for _, line := range admissible() {
		if strings.Contains(line, text) {
			removed++
			continue
		}
		out = append(out, line)
	}
	if removed != 1 {
		t.Fatalf("the fixture holds %d line(s) carrying %q; the case would not be testing what it says", removed, text)
	}
	return joined(out)
}

func TestCapabilityAbsenceAcceptsTheAdmissibleJob(t *testing.T) {
	o := capabilityAbsenceLeg(absenceTree(t, joined(admissible())))

	if o.verdict != passed {
		t.Fatalf("the admissible job was refused: %s", o.detail)
	}
	if !strings.Contains(o.detail, "4 absence(s)") {
		t.Errorf("the pass does not say how many absences it read:\n%s", o.detail)
	}
}

// The whole point of the leg. A step deleted leaves a job that goes on passing
// while proving one thing fewer, and no other check in this tree notices.
func TestCapabilityAbsenceRefusesAJobThatStoppedEstablishingOne(t *testing.T) {
	for _, c := range capabilitySteps() {
		t.Run(c.capability, func(t *testing.T) {
			o := capabilityAbsenceLeg(absenceTree(t, without(t, c.step)))

			if o.verdict != failed {
				t.Fatalf("a job with no step for %s passed: %v %s", c.capability, o.verdict, o.detail)
			}
			for _, want := range []string{c.step, c.capability} {
				if !strings.Contains(o.detail, want) {
					t.Errorf("the failure does not name %q:\n%s", want, o.detail)
				}
			}
		})
	}
}

func TestCapabilityAbsenceRefusesAJobThatReportsNoResidue(t *testing.T) {
	o := capabilityAbsenceLeg(absenceTree(t, without(t, residueStep)))

	if o.verdict != failed {
		t.Fatalf("a job that never says what it could not establish passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, residueStep) {
		t.Errorf("the failure does not name the missing step:\n%s", o.detail)
	}
}

func TestCapabilityAbsenceRefusesAJobThatRunsNoTier(t *testing.T) {
	o := capabilityAbsenceLeg(absenceTree(t, without(t, tierStep)))

	if o.verdict != failed {
		t.Fatalf("a job that establishes four absences and runs nothing inside them passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, tierStep) {
		t.Errorf("the failure does not name the missing step:\n%s", o.detail)
	}
}

// The residue is worth most on the run that went red, and a step after a failing
// one does not run at all.
func TestCapabilityAbsenceRefusesTheResidueReportedAfterTheVerdict(t *testing.T) {
	lines := admissible()
	moved := append([]string{}, lines[:len(lines)-4]...)
	moved = append(moved, lines[len(lines)-2:]...)
	moved = append(moved, lines[len(lines)-4:len(lines)-2]...)

	o := capabilityAbsenceLeg(absenceTree(t, joined(moved)))

	if o.verdict != failed {
		t.Fatalf("a job printing the residue after the tier passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "does not run at all") {
		t.Errorf("the failure does not say why the order matters:\n%s", o.detail)
	}
}

// This is #83's fifth condition. The gate tier is measured once, and a second
// measurement of the same statements against the same floor is the tier counted
// twice.
func TestCapabilityAbsenceRefusesAJobThatMeasuresCoverage(t *testing.T) {
	for _, flag := range coverageFlags {
		t.Run(flag, func(t *testing.T) {
			body := strings.Replace(joined(admissible()), "run: go test ./...", "run: go test "+flag+" ./...", 1)

			o := capabilityAbsenceLeg(absenceTree(t, body))

			if o.verdict != failed {
				t.Fatalf("a job passing %s passed the leg: %v %s", flag, o.verdict, o.detail)
			}
			for _, want := range []string{flag, "docs/coverage-floor.md", "counted twice"} {
				if !strings.Contains(o.detail, want) {
					t.Errorf("the failure does not carry %q:\n%s", want, o.detail)
				}
			}
		})
	}
}

func TestCapabilityAbsenceRefusesAJobThatRunsAnotherTier(t *testing.T) {
	body := strings.Replace(joined(admissible()), "run: go test ./...", "run: go test -tags integration ./...", 1)

	o := capabilityAbsenceLeg(absenceTree(t, body))

	if o.verdict != failed {
		t.Fatalf("a job compiling a second tier passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "-tags") {
		t.Errorf("the failure does not name the flag:\n%s", o.detail)
	}
}

func TestCapabilityAbsenceRefusesAJobThatRunsOnNoPullRequest(t *testing.T) {
	o := capabilityAbsenceLeg(absenceTree(t, without(t, "pull_request:")))

	if o.verdict != failed {
		t.Fatalf("a job that never runs on a pull request passed: %v %s", o.verdict, o.detail)
	}
	if !strings.Contains(o.detail, "pull_request") {
		t.Errorf("the failure does not name the missing trigger:\n%s", o.detail)
	}
}

// A file that is not there is the loudest way this can rot, and it is the one a
// leg reading a single path has to say something useful about.
func TestCapabilityAbsenceRefusesATreeWithNoSuchJob(t *testing.T) {
	o := capabilityAbsenceLeg(t.TempDir())

	if o.verdict != failed {
		t.Fatalf("a tree with no such workflow passed: %v %s", o.verdict, o.detail)
	}
	for _, want := range []string{absenceWorkflow, "#83"} {
		if !strings.Contains(o.detail, want) {
			t.Errorf("the failure does not carry %q:\n%s", want, o.detail)
		}
	}
}

// A comment is not a step and not a flag. The real file argues in its own
// comments about coverage and about build tags, and a leg reading them as
// declarations would refuse the file for explaining itself.
func TestCapabilityAbsenceReadsNoComment(t *testing.T) {
	lines := admissible()
	commented := append([]string{
		"# This job passes no -cover and no -tags, and the two sentences above",
		"# are why. name: Run the gate tier is written here as prose.",
	}, lines...)

	if o := capabilityAbsenceLeg(absenceTree(t, joined(commented))); o.verdict != passed {
		t.Fatalf("a comment naming a flag was read as one: %s", o.detail)
	}
}

// The leg has to judge the file this repository actually ships, not only the
// fixtures beside it. A pass here is the statement that the shipped job still
// establishes what #83 asks of it.
func TestTheJobThisRepositoryShipsIsAdmissible(t *testing.T) {
	if o := capabilityAbsenceLeg(".."); o.verdict != passed {
		t.Fatalf("%s as it stands in this tree is refused by its own leg:\n%s", absenceWorkflow, o.detail)
	}
}
