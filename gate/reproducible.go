package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// A project whose subject is reproducibility should be able to reproduce its
// own binary, and the way that stops being true is not a decision anybody
// takes. It is a build flag nobody passed, a timestamp somebody embedded, or a
// path from one machine baked into an artefact every other machine then fails
// to match. None of the three is visible in a diff, so this leg builds the
// command twice and compares what came out.
//
// What it proves is the same TREE twice rather than the same COMMIT twice, and
// the difference is not pedantry. The toolchain records vcs.modified in the
// binary, which is a fact about the working directory rather than about the
// commit, so one clean checkout and one dirty checkout of one commit differ by
// construction. Pinning the tree state is outside what a leg can do, so the
// narrower claim is the one made, and it is the claim that catches a build
// which embeds a clock or a hostname.
//
// The absolute path half is here rather than in its own leg because it is a
// property of the same two builds and needs no third one. -trimpath is what
// removes the build directory from the artefact, and forgetting it produces no
// output at all: the binary is simply 6656 bytes larger and carries somebody's
// home directory. Searching for the directory the build ran in is the whole
// evidence that the flag was passed.

// buildFlags is the flag set both builds below use, and it is the flag set the
// released artefact has to be built with for its checksum to mean anything. It
// is one value rather than two literals so that a change here is a change to
// what the leg proves.
var buildFlags = []string{"-trimpath"}

// buildFunc compiles the command rooted at root into the file out. It is a
// parameter rather than a call because this leg's own test is a gate tier test,
// and record 0009 refuses os/exec there: the test drives the leg with bytes it
// chose instead of with a compiler.
type buildFunc func(root, out string) error

// goBuild is the real one. Nothing tests it, for the reason above, and it is
// kept to one line of behaviour so there is little in it to be wrong.
func goBuild(root, out string) error {
	args := append(append([]string{"build"}, buildFlags...), "-o", out, ".")
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	if combined, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go %s: %v\n%s", strings.Join(args, " "), err, strings.TrimRight(string(combined), "\r\n"))
	}
	return nil
}

// reproducibleBuildLeg arranges the two builds and hands the judging to
// reproducibleBuild. Where there is no toolchain it reports a skip naming what
// was missing, because a leg that examined nothing and a leg that found nothing
// are different statements.
func reproducibleBuildLeg(root string) outcome {
	if _, err := exec.LookPath("go"); err != nil {
		return skip("no `go` on PATH, so nothing on this run built the command twice or compared what came out. " +
			"Asking for it costs a Go toolchain at the version go.mod declares and a writable temporary directory.")
	}
	dir, err := os.MkdirTemp("", "gate-reproducible-")
	if err != nil {
		return fail("could not make a directory to build into: " + err.Error())
	}
	defer os.RemoveAll(dir)
	return reproducibleBuild(root, dir, goBuild)
}

// reproducibleBuild builds twice into workdir and judges the two artefacts. Its
// verdict rests on bytes it read from disk rather than on what the builder
// reported, so a builder that succeeded and wrote nothing is a failure here and
// not a pass.
func reproducibleBuild(root, workdir string, build buildFunc) outcome {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return fail("could not resolve the build directory: " + err.Error())
	}

	first, err := buildOnce(root, filepath.Join(workdir, "first"), build)
	if err != nil {
		return fail("the first build did not produce an artefact to compare:\n" + err.Error())
	}
	second, err := buildOnce(root, filepath.Join(workdir, "second"), build)
	if err != nil {
		return fail("the second build did not produce an artefact to compare:\n" + err.Error())
	}

	if !bytes.Equal(first, second) {
		return fail(fmt.Sprintf(
			"two builds of this tree produced different bytes.\n"+
				"    first   %s  %d bytes\n"+
				"    second  %s  %d bytes\n\n"+
				"Something in the build reads a value that moves: a timestamp, a build host, a\n"+
				"path, or a source of randomness. A binary whose checksum moves between two builds\n"+
				"of one tree cannot be checked against a published checksum by anybody.",
			sum(first), len(first), sum(second), len(second)))
	}

	if leaked := pathsFoundIn(first, absolute); len(leaked) > 0 {
		return fail(fmt.Sprintf(
			"the built binary carries the directory it was built in:\n    %s\n\n"+
				"Pass -trimpath. Without it the artefact holds the absolute paths of the machine\n"+
				"that made it, which differ per machine, so no two people can produce the same\n"+
				"bytes and the checksum above proves nothing outside this directory. The flag\n"+
				"produces no output when it is forgotten, so this search is the whole evidence\n"+
				"that it was passed.",
			strings.Join(leaked, "\n    ")))
	}

	return note(fmt.Sprintf(
		"two builds produced the same %d bytes, sha256 %s, and neither carries %s. "+
			"That is the same tree twice rather than the same commit twice: the toolchain records "+
			"whether the working tree was modified, which is not a property of the commit.",
		len(first), sum(first), filepath.ToSlash(absolute)))
}

// buildOnce runs the builder and reads back what it wrote. The read is the
// point: an artefact nobody can open is not an artefact anybody can compare.
func buildOnce(root, out string, build buildFunc) ([]byte, error) {
	if err := build(root, out); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(out)
	if err != nil {
		return nil, fmt.Errorf("the build reported success and left nothing readable at %s: %w", filepath.ToSlash(out), err)
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("the build reported success and left an empty file at %s", filepath.ToSlash(out))
	}
	return b, nil
}

// pathsFoundIn looks for the build directory in the artefact, in both
// spellings, because the tree is developed on Windows and built on Linux and
// the toolchain writes the separator of the machine it ran on.
//
// The match is containment rather than equality, so a path UNDER the build
// directory is caught too, and those are build machine paths for the same
// reason. What that costs is a directory whose name merely begins with the
// build directory's name, which would be reported as a leak. Nothing in a Go
// binary produces one, and the narrower rule would need a path parser over
// bytes that are not known to be paths.
func pathsFoundIn(artefact []byte, absolute string) []string {
	var found []string
	for _, spelling := range []string{filepath.ToSlash(absolute), strings.ReplaceAll(filepath.ToSlash(absolute), "/", `\`)} {
		if bytes.Contains(artefact, []byte(spelling)) && !contains(found, spelling) {
			found = append(found, spelling)
		}
	}
	return found
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
