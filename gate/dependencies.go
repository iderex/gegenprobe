package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Three ways a dependency surface goes wrong without anybody deciding it should,
// and this leg refuses all three.
//
// A module resolving to bytes other than the ones recorded. `go mod verify` is
// what answers that, and its answer is taken from its exit status rather than
// from its wording, so a change to the toolchain's message cannot turn this leg
// into one that always passes.
//
// A build that quietly rewrote go.mod or go.sum. That is the lockfile drift the
// target board checks for, and it is caught here by reading both files, building,
// and reading them again. Nothing about the working tree's git state is involved,
// so the leg says the same thing in a checkout with uncommitted work as in a
// clean one.
//
// A dependency arriving with no argument behind it. Record 0001 asks for a small
// surface and docs/dependencies.md carries the number and one section per
// dependency; this leg refuses a go.mod that disagrees with either.

// dependencyDoc holds the budget and the reasons. It is a document rather than a
// constant here because the number is an argument somebody has to make.
const dependencyDoc = "docs/dependencies.md"

var (
	budgetLine    = regexp.MustCompile(`(?m)^Direct dependencies:\s*(\d+)\s*$`)
	reasonHeading = regexp.MustCompile(`(?m)^##\s+(\S+/\S+)\s*$`)
	requireBlock  = regexp.MustCompile(`(?ms)^require\s*\(\s*(.*?)^\)`)
	requireSingle = regexp.MustCompile(`(?m)^require\s+(\S+)\s+\S+\s*(//.*)?$`)
)

// moduleFiles is what a build is not allowed to change.
type moduleFiles struct {
	gomod []byte
	gosum []byte
}

// directRequires reads the module paths go.mod requires directly. An indirect
// requirement is not one this repository chose, so it is not in the budget and
// not owed a reason.
func directRequires(gomod []byte) []string {
	var found []string
	text := strings.ReplaceAll(string(gomod), "\r\n", "\n")

	for _, block := range requireBlock.FindAllStringSubmatch(text, -1) {
		for _, line := range strings.Split(block[1], "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "//") || strings.Contains(line, "// indirect") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				found = append(found, fields[0])
			}
		}
	}
	for _, m := range requireSingle.FindAllStringSubmatch(text, -1) {
		if strings.Contains(m[2], "indirect") {
			continue
		}
		found = append(found, m[1])
	}

	sort.Strings(found)
	return found
}

// budgetFrom reads the declared number and the modules the document gives a
// reason for.
func budgetFrom(doc []byte) (int, []string, error) {
	m := budgetLine.FindSubmatch(doc)
	if m == nil {
		return 0, nil, fmt.Errorf("%s declares no `Direct dependencies:` number, so there is no budget to check against", dependencyDoc)
	}
	declared, err := strconv.Atoi(string(m[1]))
	if err != nil {
		return 0, nil, fmt.Errorf("%s declares a budget that is not a number: %v", dependencyDoc, err)
	}
	var reasoned []string
	for _, h := range reasonHeading.FindAllSubmatch(doc, -1) {
		reasoned = append(reasoned, string(h[1]))
	}
	sort.Strings(reasoned)
	return declared, reasoned, nil
}

// judgeBudget compares what the module file requires against what the document
// declares, in both directions. One direction catches a dependency nobody
// argued for; the other catches a reason left behind for something that has
// gone, which is how a register stops describing the tree.
func judgeBudget(gomod, doc []byte) outcome {
	required := directRequires(gomod)
	declared, reasoned, err := budgetFrom(doc)
	if err != nil {
		return fail(err.Error())
	}

	if len(required) != declared {
		return fail(fmt.Sprintf(
			"go.mod requires %d direct dependenc(ies) and %s declares a budget of %d.\n    required: %s\n\n"+
				"Adding one is an edit to that number in the same change, with the reason beside it.",
			len(required), dependencyDoc, declared, listOrNone(required)))
	}

	if missing := notIn(required, reasoned); len(missing) > 0 {
		return fail(fmt.Sprintf(
			"go.mod requires %s and %s gives no reason for it.\n\n"+
				"A dependency with no argument behind it is a convenience rather than a decision.",
			strings.Join(missing, ", "), dependencyDoc))
	}
	if stale := notIn(reasoned, required); len(stale) > 0 {
		return fail(fmt.Sprintf(
			"%s gives a reason for %s, which go.mod does not require.\n\n"+
				"A register that outlives what it describes stops being read as one.",
			dependencyDoc, strings.Join(stale, ", ")))
	}

	return pass()
}

// judgeVerify turns the checksum verification into a verdict. The judgement is
// on the exit status rather than on the words, so a change to what the toolchain
// prints cannot silently turn this into a leg that always passes.
func judgeVerify(output string, err error) outcome {
	if err == nil {
		return pass()
	}
	return fail("module checksums do not match what is recorded:\n" +
		strings.TrimRight(output, "\r\n") + "\n\ngo mod verify: " + err.Error())
}

// judgeDrift refuses a build that rewrote the module files. A build that
// resolved something other than what was pinned leaves no other trace.
func judgeDrift(before, after moduleFiles) outcome {
	var moved []string
	if !bytes.Equal(before.gomod, after.gomod) {
		moved = append(moved, "go.mod")
	}
	if !bytes.Equal(before.gosum, after.gosum) {
		moved = append(moved, "go.sum")
	}
	if len(moved) == 0 {
		return pass()
	}
	return fail(fmt.Sprintf(
		"building rewrote %s.\n\n"+
			"The build resolved something other than what the module files pinned, and it left no "+
			"other trace of having done so. Run the build, commit what it wrote, and say in the "+
			"change what moved.",
		strings.Join(moved, " and ")))
}

// dependenciesLeg is the three judgements in order, cheapest first.
func dependenciesLeg(root string) outcome {
	gomod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return fail("could not read go.mod: " + err.Error())
	}
	doc, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(dependencyDoc)))
	if err != nil {
		return fail("could not read the dependency budget: " + err.Error())
	}
	if o := judgeBudget(gomod, doc); o.verdict != passed {
		return o
	}

	if _, err := exec.LookPath("go"); err != nil {
		return skip("the budget in " + dependencyDoc + " matches go.mod, but no `go` is on PATH, so nothing on this " +
			"run verified module checksums or watched a build for module file drift. " +
			"Asking for it costs a Go toolchain at the version go.mod declares.")
	}

	before := readModuleFiles(root)
	if o := judgeVerify(runAt(root, "go", "mod", "verify")); o.verdict != passed {
		return o
	}
	if o := command(root, "go", "build", "./..."); o.verdict != passed {
		return o
	}
	if o := judgeDrift(before, readModuleFiles(root)); o.verdict != passed {
		return o
	}

	required := directRequires(gomod)
	if len(required) == 0 {
		return note("0 direct dependencies, which is what " + dependencyDoc + " declares. " +
			"`go mod verify` therefore verified an empty set rather than finding nothing wrong with a full one, " +
			"and a build left go.mod and go.sum untouched.")
	}
	return note(fmt.Sprintf("%d direct dependenc(ies), each with a reason in %s, checksums verified, and a build left go.mod and go.sum untouched.",
		len(required), dependencyDoc))
}

// readModuleFiles reads both, treating an absent go.sum as empty. A tree with no
// dependencies has none, and a build that creates one is drift the same way a
// build that edits one is.
func readModuleFiles(root string) moduleFiles {
	gomod, _ := os.ReadFile(filepath.Join(root, "go.mod"))
	gosum, _ := os.ReadFile(filepath.Join(root, "go.sum"))
	return moduleFiles{gomod: gomod, gosum: gosum}
}

func runAt(root, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// notIn returns the entries of a that b does not hold. Both are sorted, and the
// result is what one register is missing against the other.
func notIn(a, b []string) []string {
	var out []string
	for _, x := range a {
		if !contains(b, x) {
			out = append(out, x)
		}
	}
	return out
}

func listOrNone(entries []string) string {
	if len(entries) == 0 {
		return "none"
	}
	return strings.Join(entries, ", ")
}
