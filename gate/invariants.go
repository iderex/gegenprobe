package main

import (
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// This leg holds the rules about this tree that are cheap to state as a pattern
// and expensive to rediscover by review. Each one comes from a decision record
// that already argued it, and the refusal names that record rather than the
// pattern, so somebody who trips a rule is sent to the argument instead of to a
// regular expression.
//
// Two of the rules count packages rather than judge a file: a property of the
// shape "exactly one place in the tree does this" cannot be decided by reading
// one file, and stating it as an exemption for a named package would mean
// choosing that package's path here, before the issue that creates it has. So
// the rule refuses the second site instead of the wrong one, and the first
// change to do the thing at all lands without touching this file.
//
// What the leg reads is source. It cannot see a map arriving through a struct
// field, a conversion performed by a package it cannot name, or a rule whose
// subject nobody has written yet, and where a rule found nothing to judge the
// run says so rather than passing in silence.
const (
	ruleCaseParse = "case-parsed-once"
	ruleConstants = "one-constant-table"
	ruleRenderer  = "renderer-reads-the-bundle"
	ruleNoVerdict = "no-average-no-ordering"
	ruleSorted    = "sorted-output"
	ruleTierTag   = "tier-named-by-its-tag"
)

// invariant is one rule. The record is where the rule was decided, held as a
// path this leg resolves against the tree, so a rule pointing at a record that
// was renamed or never landed reddens rather than sending a contributor
// nowhere.
type invariant struct {
	id     string
	record string
	says   string
}

// invariantList is the set, in the order the leg reports them. Every entry has a
// fixture in the suite that trips it and no other, and a test asserts this list
// and that set cover each other, so a rule cannot arrive without proof that it
// bites.
func invariantList() []invariant {
	return []invariant{
		{
			id:     ruleCaseParse,
			record: "docs/decisions/0002-the-case-file.md",
			says:   "the case file is read once, by the loader, and everything after it reads the canonical form and never the YAML",
		},
		{
			id:     ruleConstants,
			record: "docs/decisions/0004-the-common-data-model.md",
			says:   "the constant values live in exactly one table in the source and no other place in the tree holds a copy",
		},
		{
			id:     ruleRenderer,
			record: "docs/decisions/0007-bundle-and-report.md",
			says:   "the report is a rendering of the bundle, and a consumer that wants to know what a run concluded reads the bundle",
		},
		{
			id:     ruleNoVerdict,
			record: "docs/decisions/0006-what-agreement-means.md",
			says:   "the harness does not average across codes, does not put them in an order and declares no winner",
		},
		{
			id:     ruleSorted,
			record: "docs/decisions/0008-determinism-and-significant-digits.md",
			says:   "sorted iteration everywhere output is produced, and no reliance on map ordering",
		},
		{
			id:     ruleTierTag,
			record: "docs/decisions/0009-three-test-tiers.md",
			says:   "a file with no build tag is a gate test, so a file belonging to another tier has to say so",
		},
	}
}

func invariantIDs() []string {
	out := make([]string, 0, len(invariantList()))
	for _, inv := range invariantList() {
		out = append(out, inv.id)
	}
	return out
}

func invariantByID(id string) invariant {
	for _, inv := range invariantList() {
		if inv.id == id {
			return inv
		}
	}
	return invariant{id: id}
}

// breach is one refused site. The rule is carried beside the words so the suite
// can assert which rule fired without reading the sentence.
type breach struct {
	file string
	line int
	rule string
	why  string
}

// verdictWords are the function names decision 0006 refuses outright. They are
// compared as whole words after splitting an identifier, so Meaning is not mean
// and Rankine is not rank. The list is short on purpose: a longer one would
// refuse a legitimate name and the answer to that is a narrower rule rather than
// a suppression, which is what record 0009 says about a denylist that overreaches.
var verdictWords = map[string]bool{
	"average": true, "averages": true, "averaged": true,
	"mean": true, "means": true, "median": true,
	"rank": true, "ranks": true, "ranked": true, "ranking": true, "rankings": true,
	"winner": true, "winners": true,
}

// renderingPackages are the package names decision 0007's renderer will carry
// whichever path it is placed at. Naming the package rather than the directory
// is what keeps this rule from depending on a path nobody has chosen yet.
var renderingPackages = map[string]bool{"report": true, "render": true, "renderer": true}

// readingCalls reach the filesystem. A renderer that reaches it can put in the
// report something the bundle does not carry, which is the statement decision
// 0007 makes about what a consumer can trust.
var readingCalls = map[string]bool{
	"ReadFile": true, "Open": true, "OpenFile": true, "ReadDir": true, "Stat": true, "Lstat": true,
}

// writingCalls are how output leaves a function. A call to one of these inside a
// range over a map is output in map order, which is the thing decision 0008
// refuses by name.
var writingCalls = map[string]bool{
	"Fprint": true, "Fprintf": true, "Fprintln": true, "WriteString": true, "Write": true, "Encode": true,
}

// tierTags are the tags record 0009 gives the two tiers that are not the gate.
var tierTags = []string{"integration", "regression"}

// significantDigitsInATable is the count above which a floating point literal is
// a measured constant rather than a threshold somebody typed. Decision 0004 puts
// the constant values in exactly one table in the source; a tolerance, a version
// or a percentage is not one of them and does not carry this many digits.
const significantDigitsInATable = 10

// invariantsLeg refuses source that trips a rule a decision record already
// fixed.
func invariantsLeg(root string) outcome {
	var unresolved []string
	for _, inv := range invariantList() {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(inv.record))); err != nil {
			unresolved = append(unresolved, fmt.Sprintf("%s names %s, and there is no such record in this tree", inv.id, inv.record))
		}
	}
	if len(unresolved) > 0 {
		return fail(strings.Join(unresolved, "\n") +
			"\n\nA rule sends whoever trips it to the argument behind it. One naming a" +
			"\nrecord that is not there sends them nowhere, so the record is part of the" +
			"\nrule rather than a reference beside it.")
	}

	files, err := goSources(root)
	if err != nil {
		return fail(err.Error())
	}
	if len(files) == 0 {
		return skip("there is no Go source in this tree, so nothing was read")
	}

	found, judged, err := judgeSources(files, enabled(invariantIDs()))
	if err != nil {
		return fail(err.Error())
	}

	if len(found) > 0 {
		lines := make([]string, 0, len(found))
		for _, b := range found {
			inv := invariantByID(b.rule)
			lines = append(lines, fmt.Sprintf("%s:%d: %s (%s)\n    %s says %s", b.file, b.line, b.why, b.rule, inv.record, inv.says))
		}
		return fail(strings.Join(lines, "\n") +
			"\n\nEach rule here comes from a record that argued it. Where the rule is wrong" +
			"\nrather than the code, the record is what has to move, and a change that" +
			"\nsupersedes one is how that is done.")
	}

	var idle []string
	for _, id := range invariantIDs() {
		if judged[id] == 0 {
			idle = append(idle, id)
		}
	}
	summary := fmt.Sprintf("%d Go file(s) read against %d rule(s).", len(files), len(invariantList()))
	if len(idle) > 0 {
		summary += fmt.Sprintf(" %d rule(s) found nothing in this tree to judge: %s. A rule with no subject here is proven by its fixture and by nothing that was written, so this run covers less than the rule list does.",
			len(idle), strings.Join(idle, ", "))
	}
	return note(summary)
}

// goSources reads every Go file in the tree, keyed by its slash separated path.
// Source under a testdata directory is not compiled by the go tool, so it is not
// part of this module and is not judged.
func goSources(root string) (map[string]string, error) {
	files := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "testdata" {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(p) != ".go" {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(src)
		return nil
	})
	return files, err
}

// judgeSources reads a whole tree at once, because two of the rules are about
// how many packages do a thing and neither can be decided from one file. The
// second result counts, per rule, how many sites of the kind it judges were
// read, so a rule that refused nothing because it saw nothing is distinguishable
// from one that refused nothing because the tree is clean.
func judgeSources(files map[string]string, on map[string]bool) ([]breach, map[string]int, error) {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var found []breach
	judged := map[string]int{}
	var yamlSites, constantSites []breach

	for _, p := range paths {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, p, files[p], parser.ParseComments)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", p, err)
		}
		at := func(pos token.Pos) int { return fset.Position(pos).Line }

		if strings.HasSuffix(path.Base(p), "_test.go") {
			if on[ruleTierTag] {
				judged[ruleTierTag] += tierCandidates(p)
				found = append(found, tierBreaches(p, f)...)
			}
			continue
		}

		if on[ruleCaseParse] {
			for _, s := range yamlImports(p, f, at) {
				judged[ruleCaseParse]++
				yamlSites = append(yamlSites, s)
			}
		}
		if on[ruleConstants] {
			for _, s := range longFloats(p, f, at) {
				judged[ruleConstants]++
				constantSites = append(constantSites, s)
			}
		}
		if on[ruleRenderer] && renderingPackages[f.Name.Name] {
			judged[ruleRenderer]++
			found = append(found, rendererReads(p, f, at)...)
		}
		if on[ruleNoVerdict] {
			n, bs := verdictFunctions(p, f, at)
			judged[ruleNoVerdict] += n
			found = append(found, bs...)
		}
		if on[ruleSorted] {
			n, bs := mapOrderedOutput(p, f, at)
			judged[ruleSorted] += n
			found = append(found, bs...)
		}
	}

	found = append(found, secondPackage(yamlSites)...)
	found = append(found, secondPackage(constantSites)...)

	sort.Slice(found, func(i, j int) bool {
		if found[i].file != found[j].file {
			return found[i].file < found[j].file
		}
		return found[i].line < found[j].line
	})
	return found, judged, nil
}

// secondPackage returns every site of a kind that more than one package
// performs. Where one package does it the property holds and nothing is refused;
// where two do, both are named, because which of them is the one that should
// have kept it is not this leg's judgement.
func secondPackage(sites []breach) []breach {
	dirs := map[string]bool{}
	for _, s := range sites {
		dirs[path.Dir(s.file)] = true
	}
	if len(dirs) < 2 {
		return nil
	}
	names := make([]string, 0, len(dirs))
	for d := range dirs {
		names = append(names, d)
	}
	sort.Strings(names)

	out := make([]breach, 0, len(sites))
	for _, s := range sites {
		s.why = fmt.Sprintf("%s, and %d packages in this tree do it: %s", s.why, len(names), strings.Join(names, ", "))
		out = append(out, s)
	}
	return out
}

// yamlImports finds the sites that decode the case file's own format. Everything
// after the loader reads the canonical form, so a second package reaching for a
// YAML decoder is a second opinion about what a case means.
func yamlImports(p string, f *ast.File, at func(token.Pos) int) []breach {
	var out []breach
	for _, spec := range f.Imports {
		imported, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		if !namesYAML(imported) {
			continue
		}
		out = append(out, breach{
			file: p, line: at(spec.Pos()), rule: ruleCaseParse,
			why: "imports " + imported + ", which decodes the case file's own format",
		})
	}
	return out
}

func namesYAML(importPath string) bool {
	for _, segment := range strings.Split(importPath, "/") {
		if strings.Contains(strings.ToLower(segment), "yaml") {
			return true
		}
	}
	return false
}

// longFloats finds the numeric literals that are measured constants rather than
// thresholds. Decision 0004 keeps them in one table carrying the identifier each
// was taken from, and a copy elsewhere is a number that stops moving when the
// table does.
func longFloats(p string, f *ast.File, at func(token.Pos) int) []breach {
	var out []breach
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.FLOAT {
			return true
		}
		digits := significantDigits(lit.Value)
		if digits < significantDigitsInATable {
			return true
		}
		out = append(out, breach{
			file: p, line: at(lit.Pos()), rule: ruleConstants,
			why: fmt.Sprintf("holds a %d digit numeric constant", digits),
		})
		return true
	})
	return out
}

// significantDigits counts the mantissa's digits from the leading non zero one,
// which is the count decision 0008 defines for a number with a decimal point and
// is the one that separates a measured constant from a tolerance.
func significantDigits(literal string) int {
	mantissa, _, _ := strings.Cut(strings.ToLower(literal), "e")
	mantissa = strings.ReplaceAll(strings.ReplaceAll(mantissa, "_", ""), ".", "")
	mantissa = strings.TrimLeft(mantissa, "0")
	count := 0
	for _, r := range mantissa {
		if r >= '0' && r <= '9' {
			count++
		}
	}
	return count
}

// rendererReads refuses a filesystem read in the package that renders the
// report. What the report says has to be in the bundle, because the bundle is
// what a consumer keeps and what the checksum covers.
func rendererReads(p string, f *ast.File, at func(token.Pos) int) []breach {
	var out []breach
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		pkg, name, ok := qualifiedCall(call)
		if !ok || pkg != "os" || !readingCalls[name] {
			return true
		}
		out = append(out, breach{
			file: p, line: at(call.Pos()), rule: ruleRenderer,
			why: "calls os." + name + " in the package that renders the report",
		})
		return true
	})
	return out
}

// verdictFunctions refuses a function whose name says it produces a verdict.
// The count returned is every function read, because this rule judges the whole
// declaration set rather than a candidate subset of it.
func verdictFunctions(p string, f *ast.File, at func(token.Pos) int) (int, []breach) {
	read := 0
	var out []breach
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		read++
		for _, w := range identifierWords(fn.Name.Name) {
			if !verdictWords[w] {
				continue
			}
			out = append(out, breach{
				file: p, line: at(fn.Pos()), rule: ruleNoVerdict,
				why: "declares " + fn.Name.Name + ", whose name says it " + w + "s the codes",
			})
			break
		}
	}
	return read, out
}

// identifierWords splits a Go identifier into lower case words, so a rule about
// a word is not a rule about a substring. Meaning is not mean and Rankine is not
// rank.
func identifierWords(name string) []string {
	var words []string
	current := strings.Builder{}
	for _, r := range name {
		switch {
		case r == '_':
			if current.Len() > 0 {
				words = append(words, strings.ToLower(current.String()))
				current.Reset()
			}
		case r >= 'A' && r <= 'Z':
			if current.Len() > 0 {
				words = append(words, strings.ToLower(current.String()))
				current.Reset()
			}
			current.WriteRune(r)
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		words = append(words, strings.ToLower(current.String()))
	}
	return words
}

// mapOrderedOutput refuses a write performed inside a range over a map. The
// count returned is every range statement read.
//
// What it sees is a map it can name in the same file: one written as a literal,
// made, declared, or arriving as a parameter or a result. A map reached through
// a struct field or through another package's return type is not seen, so this
// is a floor rather than a guarantee, and collecting keys and sorting them
// before the write is what it is asking for rather than something it forbids.
func mapOrderedOutput(p string, f *ast.File, at func(token.Pos) int) (int, []breach) {
	maps := mapIdentifiers(f)
	read := 0
	var out []breach

	ast.Inspect(f, func(n ast.Node) bool {
		rng, ok := n.(*ast.RangeStmt)
		if !ok {
			return true
		}
		read++
		if !rangesOverAMap(rng.X, maps) {
			return true
		}
		ast.Inspect(rng.Body, func(inner ast.Node) bool {
			call, ok := inner.(*ast.CallExpr)
			if !ok {
				return true
			}
			_, name, ok := qualifiedCall(call)
			if !ok || !writingCalls[name] {
				return true
			}
			out = append(out, breach{
				file: p, line: at(call.Pos()), rule: ruleSorted,
				why: "calls " + name + " inside a range over a map, so what it writes is in map order",
			})
			return true
		})
		return true
	})
	return read, out
}

func rangesOverAMap(x ast.Expr, maps map[string]bool) bool {
	switch e := x.(type) {
	case *ast.Ident:
		return maps[e.Name]
	case *ast.CompositeLit:
		_, ok := e.Type.(*ast.MapType)
		return ok
	}
	return false
}

// mapIdentifiers collects the names this file gives a map, from the four places
// a map is introduced with its type written out.
func mapIdentifiers(f *ast.File) map[string]bool {
	maps := map[string]bool{}

	fields := func(list *ast.FieldList) {
		if list == nil {
			return
		}
		for _, field := range list.List {
			if _, ok := field.Type.(*ast.MapType); !ok {
				continue
			}
			for _, name := range field.Names {
				maps[name.Name] = true
			}
		}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			fields(x.Recv)
			if x.Type != nil {
				fields(x.Type.Params)
				fields(x.Type.Results)
			}
		case *ast.FuncLit:
			if x.Type != nil {
				fields(x.Type.Params)
				fields(x.Type.Results)
			}
		case *ast.ValueSpec:
			if _, ok := x.Type.(*ast.MapType); !ok {
				return true
			}
			for _, name := range x.Names {
				maps[name.Name] = true
			}
		case *ast.AssignStmt:
			for i, lhs := range x.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || i >= len(x.Rhs) {
					continue
				}
				if producesAMap(x.Rhs[i]) {
					maps[ident.Name] = true
				}
			}
		}
		return true
	})
	return maps
}

func producesAMap(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.CompositeLit:
		_, ok := x.Type.(*ast.MapType)
		return ok
	case *ast.CallExpr:
		ident, ok := x.Fun.(*ast.Ident)
		if !ok || ident.Name != "make" || len(x.Args) == 0 {
			return false
		}
		_, ok = x.Args[0].(*ast.MapType)
		return ok
	}
	return false
}

// qualifiedCall returns the package or receiver name and the called name for a
// selector call, which is how os.ReadFile and w.Write are both recognised
// without resolving what w is.
func qualifiedCall(call *ast.CallExpr) (string, string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", "", false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", sel.Sel.Name, true
	}
	return ident.Name, sel.Sel.Name, true
}

// tierCandidates says whether a test file's name claims a tier, which is the
// subject of the rule below.
func tierCandidates(p string) int {
	if namedTier(p) == "" {
		return 0
	}
	return 1
}

// tierBreaches refuses a test file whose name says it belongs to another tier
// and whose build constraints leave it in the gate. Record 0009 makes the
// untagged file a gate test, so this is the one direction where a deleted line
// silently moves a test into the tier that runs everywhere.
//
// The other direction, a tagged file whose name does not say so, is not judged
// here. A tag is what the toolchain reads and a name is what a person reads, and
// only the first decides which suite a file runs in.
func tierBreaches(p string, f *ast.File) []breach {
	tier := namedTier(p)
	if tier == "" {
		return nil
	}
	if constraintTags(f)[tier] {
		return nil
	}
	return []breach{{
		file: p, line: 1, rule: ruleTierTag,
		why: "is named for the " + tier + " tier and carries no //go:build " + tier + " constraint, so the toolchain runs it in the gate",
	}}
}

func namedTier(p string) string {
	base := strings.ToLower(path.Base(p))
	for _, tag := range tierTags {
		if strings.Contains(base, tag) {
			return tag
		}
	}
	return ""
}

func constraintTags(f *ast.File) map[string]bool {
	tags := map[string]bool{}
	for _, group := range f.Comments {
		for _, c := range group.List {
			if !constraint.IsGoBuild(c.Text) {
				continue
			}
			expr, err := constraint.Parse(c.Text)
			if err != nil {
				continue
			}
			collectTags(expr, tags)
		}
	}
	return tags
}

func collectTags(e constraint.Expr, into map[string]bool) {
	switch x := e.(type) {
	case *constraint.TagExpr:
		into[x.Tag] = true
	case *constraint.NotExpr:
		collectTags(x.X, into)
	case *constraint.AndExpr:
		collectTags(x.X, into)
		collectTags(x.Y, into)
	case *constraint.OrExpr:
		collectTags(x.X, into)
		collectTags(x.Y, into)
	}
}
