package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The leg named `gate tier capabilities` reads test source. What it cannot see
// is a reach that only exists at run time, and #83 is the job that catches one:
// the same tier, on a runner where the display, the container engine, the
// elevation tool and the network are genuinely absent rather than merely
// unused. This leg is what keeps that job honest, because the job itself is
// only ever red on a pull request and a change that quietly guts it would be
// green everywhere a reader looks.
//
// Three things can rot without anybody noticing. A step that establishes an
// absence can be deleted, and the job goes on passing while proving one thing
// fewer. A coverage flag can be added, and the same tier is then counted twice
// against the floor in docs/coverage-floor.md. A build tag can be added, and
// the job stops being the gate tier in a harsher place and becomes a second
// tier nobody asked this job to run.
//
// Like the action pinning leg, this reads the file as text rather than as YAML.
// What it judges is a step name, a flag and a trigger, each written on one line
// in the file a reader edits, and a parser would be a dependency this tree does
// not carry for no property it does not already have. The cost is that a step
// name written as a folded or quoted scalar would not be seen, which this
// workflow does not do and which the failure message does not hide.

// absenceWorkflow is the file this leg reads. It is a separate workflow rather
// than a job in the main one because it is a place rather than a check: the
// checks are the tier's own, and this file only says where they run.
const absenceWorkflow = ".github/workflows/tier-without-capabilities.yml"

// capabilityStep pairs a capability #83 names with the step that has to
// establish its absence. The pairing is here rather than in the workflow so
// that removing a step is a red gate and not a quieter job.
type capabilityStep struct {
	capability string
	step       string
}

func capabilitySteps() []capabilityStep {
	return []capabilityStep{
		{"a display", "Establish that no display is set or reachable"},
		{"a container engine", "Establish that no container engine is on the path"},
		{"the network", "Establish that outbound network is refused"},
		{"an elevation tool", "Establish that no elevation tool is available"},
	}
}

const (
	// residueStep prints what the runner could not establish. It runs before
	// the tier and not after it, because a step after a failing one does not
	// run at all, and the caveat is worth most on the run that went red.
	residueStep = "Say what could not be established"

	// tierStep runs the tier itself.
	tierStep = "Run the gate tier"

	// tierRunner and tierPattern are what it runs, read as two pieces on one
	// line rather than as one string, so that a flag added between them is
	// judged by the rules below instead of turning this into a second finding
	// about a command nobody can see.
	tierRunner  = "go test"
	tierPattern = "./..."
)

// coverageFlags are the spellings that would make this job's run contribute to
// the published figure. The gate tier is measured once, by the coverage leg,
// and a second measurement of the same tier is the same statements counted
// twice. Longest first, so a line carrying -coverprofile is reported once under
// the flag it actually holds.
var coverageFlags = []string{"-coverprofile", "-covermode", "-cover"}

// capabilityAbsenceLeg refuses a #83 job that has stopped establishing
// something, started measuring coverage, or started running another tier.
func capabilityAbsenceLeg(root string) outcome {
	path := filepath.Join(root, filepath.FromSlash(absenceWorkflow))
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return fail(absenceWorkflow + " is not in this tree.\n\n" +
			"That file is the executed half of the gate tier capability rule: the leg\n" +
			"named `gate tier capabilities` reads source and cannot prove what a test\n" +
			"does at run time. Without the job, nothing does. #83 is where it is argued.")
	}
	if err != nil {
		return fail(err.Error())
	}

	lines := strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n")

	var refused []string
	refuse := func(line int, why string) {
		if line > 0 {
			refused = append(refused, fmt.Sprintf("%s:%d: %s", absenceWorkflow, line, why))
			return
		}
		refused = append(refused, absenceWorkflow+": "+why)
	}

	if !onLine(lines, "pull_request:") {
		refuse(0, "declares no pull_request trigger, so a run time reach would be found after the change landed rather than while it was still in front of somebody")
	}

	for _, c := range capabilitySteps() {
		if stepLine(lines, c.step) == 0 {
			refuse(0, "has no step named "+quote(c.step)+", so nothing in it establishes that "+c.capability+" is absent and a pass says one thing less than it appears to")
		}
	}

	residue := stepLine(lines, residueStep)
	if residue == 0 {
		refuse(0, "has no step named "+quote(residueStep)+", so a runner that could not remove a capability would produce a green tick saying nothing about which one")
	}
	tier := stepLine(lines, tierStep)
	if tier == 0 {
		refuse(0, "has no step named "+quote(tierStep)+", so it establishes four absences and then runs nothing inside them")
	}
	if residue > 0 && tier > 0 && residue > tier {
		refuse(residue, quote(residueStep)+" comes after "+quote(tierStep)+", so on the run where the tier goes red it does not run at all, which is the run the caveat is worth most on")
	}

	for i, line := range lines {
		if isComment(line) {
			continue
		}
		for _, flag := range coverageFlags {
			if strings.Contains(line, flag) {
				refuse(i+1, "passes "+flag+", which measures the gate tier a second time. The figure in docs/coverage-floor.md is that tier measured once, by the coverage leg, and the same statements counted twice is what this refuses")
				break
			}
		}
		if strings.Contains(line, "-tags") {
			refuse(i+1, "passes -tags, so this job would compile a tier other than the gate's. It is the same tier in a harsher place and is not additional coverage")
		}
	}

	if !onLineWithBoth(lines, tierRunner, tierPattern) {
		refuse(0, "runs no "+quote(tierRunner+" "+tierPattern)+", so whatever it runs is not the tier the gate runs")
	}

	if len(refused) > 0 {
		return fail(strings.Join(refused, "\n") +
			"\n\nWhat this job is and what it deliberately is not are in the comment at the\n" +
			"top of the file, and #83 is where it was argued.")
	}
	return note(fmt.Sprintf("%d absence(s) established by a named step, the residue reported before the verdict, and the gate tier run under no build tag and no coverage flag.", len(capabilitySteps())))
}

// onLine says whether any line outside a comment carries the text.
func onLine(lines []string, text string) bool {
	for _, line := range lines {
		if !isComment(line) && strings.Contains(line, text) {
			return true
		}
	}
	return false
}

// onLineWithBoth says whether one line outside a comment carries both texts.
func onLineWithBoth(lines []string, first, second string) bool {
	for _, line := range lines {
		if isComment(line) {
			continue
		}
		if strings.Contains(line, first) && strings.Contains(line, second) {
			return true
		}
	}
	return false
}

// stepLine returns the one based line a step of that name is declared on, or
// zero where no line declares one.
func stepLine(lines []string, name string) int {
	for i, line := range lines {
		if isComment(line) {
			continue
		}
		if strings.Contains(line, "name: "+name) {
			return i + 1
		}
	}
	return 0
}

func isComment(line string) bool { return strings.HasPrefix(strings.TrimSpace(line), "#") }

func quote(s string) string { return "`" + s + "`" }
