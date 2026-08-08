package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/iderex/gegenprobe/internal/fixture"
)

// The readers in this project are tested against output from fixed format
// Fortran programs, where a column position carries meaning, a trailing space
// can be significant and a carriage return is sometimes part of the record. A
// raw file does not survive a checkout: a clone made with core.autocrlf set
// rewrites every line ending in the working tree, and the fixture goes on
// passing while it tests something it was not collected to test.
//
// So a fixture is stored encoded, with its provenance in the same file, and this
// leg refuses one stored any other way. The convention is docs/fixtures.md and
// the reader that decides it is internal/fixture, which is the same code the
// tests load through, so a fixture the gate accepts is a fixture that loads.
//
// The subject is every file under a directory named testdata, and the rule is
// total over that set rather than a list of exceptions. A raw file there is
// refused whether or not its bytes happen to matter today, because which bytes
// matter is a judgement and a checkout does not make it.
const conventionDoc = "docs/fixtures.md"

// fixtureStorageLeg refuses a fixture that is not stored in the convention.
func fixtureStorageLeg(root string) outcome {
	var problems []string
	read := 0
	directories := 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git":
				return fs.SkipDir
			case "testdata":
				directories++
				n, found := judgeFixtureDirectory(path)
				read += n
				problems = append(problems, found...)
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == fixture.Extension {
			problems = append(problems, fmt.Sprintf("%s: a fixture outside a testdata directory is not where anything looks for one", filepath.ToSlash(path)))
		}
		return nil
	})
	if err != nil {
		return fail(err.Error())
	}

	if len(problems) > 0 {
		return fail(strings.Join(problems, "\n") +
			"\n\nA fixture is a provenance note, a blank line and the bytes in base64, in a" +
			"\nfile named *" + fixture.Extension + " under a testdata directory. " + conventionDoc + " says why," +
			"\nand internal/fixture.Encode renders the payload so it does not have to be" +
			"\nassembled by hand.")
	}
	if directories == 0 {
		return skip("there is no testdata directory in this tree, so no fixture was read")
	}
	if read == 0 {
		return skip(fmt.Sprintf("%d testdata directory(ies) hold no files, so no fixture was read", directories))
	}
	return note(fmt.Sprintf("%d fixture(s) read, under %d testdata directory(ies).", read, directories))
}

// judgeFixtureDirectory reads one testdata directory and everything under it. It
// returns how many fixtures it read, so that a green line stands for an
// examination rather than for an empty directory.
func judgeFixtureDirectory(dir string) (read int, problems []string) {
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := filepath.ToSlash(path)
		if filepath.Ext(path) != fixture.Extension {
			problems = append(problems, fmt.Sprintf("%s: stored raw, so a checkout is free to rewrite its bytes", name))
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		read++
		if _, found := fixture.Parse(body); len(found) > 0 {
			for _, p := range found {
				problems = append(problems, name+": "+p)
			}
		}
		return nil
	})
	if err != nil {
		problems = append(problems, err.Error())
	}
	return read, problems
}
