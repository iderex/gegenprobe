package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// A clone rewrites what it checks out, and which rewriting it does is a fact of
// somebody's local git config that no tree holds. With nothing declared, the
// bytes a contributor opens are decided by a setting nobody here can read. On
// the machine this landed from, the objects carry line feeds and every tracked
// text file in the working tree carries carriage returns, so the formatter
// reports the whole tree as unformatted and a fixture collected on one platform
// is a different file on another. Nothing about any of that looks wrong.
//
// The two legs below are what decide it instead. The first reads the
// declaration and refuses a tracked file whose type nothing in it names, so a
// new kind of file arrives as a decision rather than as a default. The second
// runs a recorded carriage return through git under that declaration and
// refuses the round trip that does not return what it was given, because a
// declaration is a claim about what git will do until something makes git do it.
const (
	attributesFile = ".gitattributes"

	// probeFixture is what the round trip is made of: a provenance note in the
	// form docs/fixtures.md fixes, and a payload whose line endings are the
	// thing being measured. It is written into a throwaway repository and
	// never into this one.
	probeFixture = "Code: hand-written\r\n" +
		"Version: not applicable\r\n" +
		"Case: not applicable\r\n" +
		"Kept: all of it\r\n" +
		"\r\n" +
		"aGVsbG8=\r\n"
)

// attributesLeg refuses a tracked file whose type no tree-wide pattern in the
// declaration names.
//
// What it does not do. It reads the patterns as text and never asks git which
// files a pattern reaches, so a pattern naming a type is taken at its word.
// That bound is the reason only a pattern with no slash in it counts: one with
// a slash reaches part of the tree, and counting it would let a rule written
// for one directory stand as a decision about the whole of it.
func attributesLeg(root string) outcome {
	declaration, err := os.ReadFile(filepath.Join(root, attributesFile))
	if os.IsNotExist(err) {
		return fail(attributesFile + " is not in this tree.\n\n" +
			"Nothing then declares what git may do to any file here, so every tracked file is\n" +
			"converted, or not, by whatever core.autocrlf is set to in whoever's clone it is.")
	}
	if err != nil {
		return fail(err.Error())
	}

	tracked, err := trackedFiles(root)
	if err != nil {
		return skip("git could not list the tracked files here, so no file type was read: " + err.Error())
	}
	if len(tracked) == 0 {
		return skip("git reports no tracked file in this tree, so no file type was read")
	}
	return judgeTypes(declaration, tracked)
}

// judgeTypes is the whole of the judgement, over a declaration and a list of
// paths rather than over a checkout, so the suite drives it with fixtures that
// differ from a passing pair by one line.
func judgeTypes(declaration []byte, tracked []string) outcome {
	named := namedTypes(declaration)

	held := map[string]int{}
	firstOfType := map[string]string{}
	var undeclared []string

	for _, p := range tracked {
		t := fileType(p)
		if held[t] == 0 {
			firstOfType[t] = p
			if !named[t] {
				undeclared = append(undeclared, t)
			}
		}
		held[t]++
	}

	if len(undeclared) > 0 {
		sort.Strings(undeclared)
		lines := make([]string, 0, len(undeclared))
		for _, t := range undeclared {
			lines = append(lines, fmt.Sprintf("%s: %d tracked file(s), the first of them %s", t, held[t], firstOfType[t]))
		}
		return fail(strings.Join(lines, "\n") +
			"\n\nNo tree-wide pattern in " + attributesFile + " names those types, so what a checkout" +
			"\ndoes to them is decided by a setting in somebody's clone. Add a line per type" +
			"\nsaying which of the two a file of it is:" +
			"\n\n    *.txt   text eol=lf   bytes a person wrote and a diff should read" +
			"\n    *.txt   -text         bytes something recorded, kept exactly as collected")
	}

	return note(fmt.Sprintf("%d tracked file(s) across %d type(s), every one named in %s.", len(tracked), len(held), attributesFile))
}

// namedTypes is the set of file types the declaration names with a tree-wide
// pattern. A macro definition, a comment and a blank line name nothing, and
// neither does a catch-all, which is what keeps a default from reading as a
// decision.
func namedTypes(declaration []byte) map[string]bool {
	named := map[string]bool{}
	for _, line := range strings.Split(strings.ReplaceAll(string(declaration), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[attr]") {
			continue
		}
		if t := typeNamedBy(strings.Fields(line)[0]); t != "" {
			named[t] = true
		}
	}
	return named
}

// typeNamedBy returns the file type one pattern names, or the empty string
// where it names none. A pattern holding a wildcard anywhere but in the
// leading "*." of an extension is not read: what such a pattern reaches is a
// question for git, and guessing at it here is how a check comes to disagree
// with the thing it is checking.
func typeNamedBy(pattern string) string {
	if strings.Contains(pattern, "/") {
		return ""
	}
	if rest, found := strings.CutPrefix(pattern, "*."); found {
		if strings.ContainsAny(rest, "*?[") || rest == "" {
			return ""
		}
		return "." + rest
	}
	if strings.ContainsAny(pattern, "*?[") {
		return ""
	}
	return pattern
}

// fileType is the extension, or the whole name where a file has none. A name
// beginning with a dot and holding no other one is its own type: .gitignore is
// not a file of type .gitignore among others, it is the only one there is, and
// a rule for it is written by naming it.
func fileType(p string) string {
	name := path.Base(p)
	if dot := strings.LastIndex(name, "."); dot > 0 {
		return name[dot:]
	}
	return name
}

// trackedFiles asks git what it is tracking. Tracked is the subject rather than
// present, because a file the tree does not carry is not something this
// repository has decided anything about, and the ignore rules beside the
// declaration are what keep build output and an operator's copy of somebody
// else's code out of that set.
func trackedFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "-z")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(bytes.TrimSpace(ee.Stderr)) > 0 {
			return nil, fmt.Errorf("git ls-files: %v: %s", err, bytes.TrimSpace(ee.Stderr))
		}
		return nil, fmt.Errorf("git ls-files: %v", err)
	}

	var paths []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths, nil
}

// recordedBytesLeg proves the declaration bites where losing is silent. It
// takes a fixture holding carriage returns through git add and a checkout,
// inside a throwaway repository carrying this tree's declaration and configured
// with core.autocrlf set, and refuses a round trip that does not hand back what
// it was given.
//
// The configuration is forced rather than inherited, so the leg asks the same
// question on every platform. A run on a machine that converts nothing would
// otherwise report a pass that was true of the machine and said nothing about
// the declaration.
//
// Nothing here touches this repository. The probe is written, staged, read back
// and thrown away in a directory of its own.
func recordedBytesLeg(root string) outcome {
	declaration, err := os.ReadFile(filepath.Join(root, attributesFile))
	if os.IsNotExist(err) {
		return skip(attributesFile + " is not in this tree, so there was no declaration to take a fixture through")
	}
	if err != nil {
		return fail(err.Error())
	}

	dir, err := os.MkdirTemp("", "gegenprobe-attributes-")
	if err != nil {
		return skip("no directory could be made to take a fixture through git in: " + err.Error())
	}
	defer os.RemoveAll(dir)

	if out, err := probeRepository(dir, declaration); err != nil {
		return skip("the round trip could not be made: " + err.Error() + strings.TrimRight("\n"+out, "\n"))
	}

	staged, err := hostileGit(dir, "cat-file", "blob", ":"+probeName)
	if err != nil {
		return skip("the staged fixture could not be read back: " + err.Error())
	}
	if diff := movedBytes("git add", []byte(probeFixture), staged); diff != "" {
		return fail(diff + repairTheDeclaration)
	}

	if err := os.Remove(filepath.Join(dir, probeName)); err != nil {
		return skip("the fixture could not be removed before checking it out again: " + err.Error())
	}
	if _, err := hostileGit(dir, "checkout", "--", probeName); err != nil {
		return skip("the fixture could not be checked out again: " + err.Error())
	}
	restored, err := os.ReadFile(filepath.Join(dir, probeName))
	if err != nil {
		return skip("the checked out fixture could not be read: " + err.Error())
	}
	if diff := movedBytes("a checkout", []byte(probeFixture), restored); diff != "" {
		return fail(diff + repairTheDeclaration)
	}

	return note(fmt.Sprintf("a %d byte fixture holding %d carriage return(s) went through git add and a checkout unchanged, "+
		"in a throwaway repository holding this tree's %s and configured with core.autocrlf=true.",
		len(probeFixture), strings.Count(probeFixture, "\r"), attributesFile))
}

// probeName is what the fixture is called inside the throwaway repository. The
// extension is the one the declaration is expected to have a rule for, because
// that rule is the subject.
const probeName = "probe.fixture"

const repairTheDeclaration = "\n\nThe rule that prevents this is a line in " + attributesFile + " marking a fixture as bytes" +
	"\nrather than as text, and every byte it protects is one a reader of a fixed format" +
	"\ntable depends on. docs/fixtures.md is where the encoding beside it is argued."

// probeRepository makes the throwaway repository, puts the declaration in it,
// writes the fixture and stages it.
func probeRepository(dir string, declaration []byte) (string, error) {
	if out, err := hostileGit(dir, "init", "-q"); err != nil {
		return string(out), err
	}
	if err := os.WriteFile(filepath.Join(dir, attributesFile), declaration, 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, probeName), []byte(probeFixture), 0o644); err != nil {
		return "", err
	}
	out, err := hostileGit(dir, "add", "--", probeName)
	return string(out), err
}

// hostileGit runs one git command with the conversion this leg exists to defeat
// switched on, whatever the machine's own configuration is. Nothing it runs
// writes outside the directory it is given.
func hostileGit(dir string, args ...string) ([]byte, error) {
	full := append([]string{"-c", "core.autocrlf=true", "-c", "core.safecrlf=false"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(bytes.TrimSpace(ee.Stderr)) > 0 {
			return out, fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, bytes.TrimSpace(ee.Stderr))
		}
		return out, fmt.Errorf("git %s: %v", strings.Join(args, " "), err)
	}
	return out, nil
}

// movedBytes says how a round trip changed a fixture, or returns the empty
// string where it did not. It reports the count of carriage returns on each
// side rather than a byte offset, because losing them is the whole failure and
// an offset would send a reader counting.
func movedBytes(stage string, want, got []byte) string {
	if bytes.Equal(want, got) {
		return ""
	}
	return fmt.Sprintf("%s did not hand back the fixture it was given: %d bytes in with %d carriage return(s), %d bytes out with %d.",
		stage, len(want), bytes.Count(want, []byte("\r")), len(got), bytes.Count(got, []byte("\r")))
}
