// Package boundary judges the import graph of this module against the direction
// docs/package-boundaries.md permits. It reads nothing: the graph is handed to
// it and the declaration is handed to it, so the judging is testable over
// recorded graphs and the only thing that has to run a process is the caller
// that asks the toolchain what the real one is.
//
// The direction is data rather than a diagram because a diagram in a document
// stops being true without anything going red. What this catches is the
// structural version of a mistake the source level lint cannot see: an edge that
// arrives through a rename, a move, or an import somebody added to the wrong
// package while fixing something else.
package boundary

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Package is one package of this module and the edges it has into the rest of
// it. Path is relative to the module, with "." for the command at the root.
// Imports come from the package's own source and TestImports from its tests,
// kept apart because a package that reads nothing and a package whose tests read
// something are different statements.
type Package struct {
	Path        string
	Imports     []string
	TestImports []string
}

// Entry is one section of the declaration: what a package may import, what its
// tests may import in addition, and the decision record the boundary comes from.
type Entry struct {
	Package     string
	Decision    string
	May         []string
	MayInTests  []string
	declaredAll map[string]bool
}

// Violation is one edge or one absence the declaration refuses. Package is the
// package it is about; Detail says what is wrong in the terms somebody repairs
// it in.
type Violation struct {
	Package string
	Detail  string
}

func (v Violation) String() string { return v.Package + ": " + v.Detail }

var (
	sectionHeading = regexp.MustCompile(`^##\s+(\S+)\s*$`)
	decisionField  = regexp.MustCompile(`^Decision:\s*(\d{4})\s*$`)
	mayField       = regexp.MustCompile(`^May-import:\s*(\S.*?)\s*$`)
	mayTestsField  = regexp.MustCompile(`^May-import-in-tests:\s*(\S.*?)\s*$`)
)

// nothing is the value that says a list is empty on purpose. An empty list
// written as an absent line and one written as "nothing" would be
// indistinguishable, and only one of them is a decision somebody made.
const nothing = "nothing"

// ParseDeclaration reads the sections out of the document. A heading whose
// section carries no Decision line is prose rather than an entry, which is what
// lets the document explain itself above and below the entries without either
// half being read as the other.
func ParseDeclaration(doc string) (map[string]Entry, error) {
	entries := map[string]Entry{}

	var current string
	var e Entry
	flush := func() error {
		if current == "" || e.Decision == "" {
			return nil
		}
		if e.May == nil || e.MayInTests == nil {
			return fmt.Errorf("the entry for %s names a decision but not both import lists", current)
		}
		if _, seen := entries[current]; seen {
			return fmt.Errorf("the document holds two entries for %s", current)
		}
		e.Package = current
		entries[current] = e
		return nil
	}

	for _, line := range strings.Split(strings.ReplaceAll(doc, "\r\n", "\n"), "\n") {
		if m := sectionHeading.FindStringSubmatch(line); m != nil {
			if err := flush(); err != nil {
				return nil, err
			}
			current, e = m[1], Entry{}
			continue
		}
		switch {
		case decisionField.MatchString(line):
			e.Decision = decisionField.FindStringSubmatch(line)[1]
		case mayField.MatchString(line):
			e.May = list(mayField.FindStringSubmatch(line)[1])
		case mayTestsField.MatchString(line):
			e.MayInTests = list(mayTestsField.FindStringSubmatch(line)[1])
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, errors.New("the declaration holds no entry, so nothing says which edge is permitted")
	}
	for path, entry := range entries {
		entry.declaredAll = map[string]bool{}
		for _, p := range entry.May {
			entry.declaredAll[p] = true
		}
		entries[path] = entry
	}
	return entries, nil
}

// list reads one comma separated field. The word nothing is an empty list and
// not a package called nothing.
func list(value string) []string {
	if value == nothing {
		return []string{}
	}
	var out []string
	for _, p := range strings.Split(value, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if out == nil {
		out = []string{}
	}
	sort.Strings(out)
	return out
}

// Conform returns every violation, in package order, so that a run over a tree
// with several says the same thing twice rather than in a different order each
// time.
func Conform(packages []Package, declaration map[string]Entry) []Violation {
	var out []Violation

	present := map[string]bool{}
	for _, p := range packages {
		present[p.Path] = true
	}

	sorted := append([]Package(nil), packages...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })

	for _, p := range sorted {
		entry, declared := declaration[p.Path]
		if !declared {
			out = append(out, Violation{
				Package: p.Path,
				Detail: "is in the tree and not in the declaration, so nothing says what it may import. " +
					"Add an entry naming the decision the boundary comes from",
			})
			continue
		}
		for _, imported := range p.Imports {
			if !entry.declaredAll[imported] {
				out = append(out, Violation{
					Package: p.Path,
					Detail:  fmt.Sprintf("imports %s, which its entry does not permit (it permits %s)", imported, describe(entry.May)),
				})
			}
		}
		permittedInTests := map[string]bool{}
		for k := range entry.declaredAll {
			permittedInTests[k] = true
		}
		for _, t := range entry.MayInTests {
			permittedInTests[t] = true
		}
		for _, imported := range p.TestImports {
			if !permittedInTests[imported] {
				out = append(out, Violation{
					Package: p.Path,
					Detail:  fmt.Sprintf("has a test importing %s, which its entry does not permit (its tests may add %s)", imported, describe(entry.MayInTests)),
				})
			}
		}
	}

	var declaredPaths []string
	for path := range declaration {
		declaredPaths = append(declaredPaths, path)
	}
	sort.Strings(declaredPaths)
	for _, path := range declaredPaths {
		if !present[path] {
			out = append(out, Violation{
				Package: path,
				Detail:  "has an entry in the declaration and is not a package in this tree, so the entry permits edges nothing can make",
			})
		}
	}
	return out
}

// describe renders a permitted list the way the document writes it, so a reader
// repairing a failure meets the same word in both places.
func describe(l []string) string {
	if len(l) == 0 {
		return nothing
	}
	return strings.Join(l, ", ")
}
