package main

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// The documentation in this project names things constantly: a decision record,
// a fixture directory, a leg of this command, the file a rule is argued in. A
// document naming a path that is not there is the quietest way documentation
// goes wrong, because nothing about it looks wrong. The three legs below judge
// the Markdown in this tree: the form it is written in, the links inside it, and
// the paths it names in prose.
//
// All three read the tree as text. A Markdown parser would decide the same
// questions more exactly and would be the first dependency this module takes,
// which docs/dependencies.md refuses without an argument for it. What that costs
// is stated at each leg rather than in one place, because each leg loses
// something different by it.
//
// The form is fixed in formDoc so that arguing about it is a change to a
// document rather than a conversation on a pull request.
const formDoc = "docs/markdown-form.md"

// documentationFormLeg refuses a Markdown file that is not in the form formDoc
// states.
//
// Like the format leg over the Go source, it judges after normalising CRLF to
// LF and says when it did. A checkout with core.autocrlf set holds every tracked
// text file with carriage returns, and a leg reporting that as a trailing
// whitespace error on every line of every document would say nothing about how
// the document is written. What the bytes in a checkout are is #24's subject and
// not this leg's.
func documentationFormLeg(root string) outcome {
	docs, err := markdownFiles(root)
	if err != nil {
		return fail(err.Error())
	}
	if len(docs) == 0 {
		return skip("there is no Markdown file in this tree, so no document was read")
	}

	var problems []string
	carriageReturns := 0

	for _, rel := range docs {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return fail(err.Error())
		}
		body := string(raw)
		if strings.Contains(body, "\r\n") {
			carriageReturns++
			body = strings.ReplaceAll(body, "\r\n", "\n")
		}
		problems = append(problems, judgeForm(rel, body)...)
	}

	if len(problems) > 0 {
		return fail(strings.Join(problems, "\n") +
			"\n\nThe form is " + formDoc + ". This leg names the line and the rule; nothing" +
			"\nrewrites the file, so the repair is an edit somebody makes and reads.")
	}
	if carriageReturns > 0 {
		return note(fmt.Sprintf("%d document(s) read, %d of them holding CRLF line endings in this checkout and judged after normalising to LF. Line endings are not this leg's subject.", len(docs), carriageReturns))
	}
	return note(fmt.Sprintf("%d document(s) read.", len(docs)))
}

// heading matches an ATX heading written the way formDoc fixes: one to six
// hashes, exactly one space, and something after it.
var heading = regexp.MustCompile(`^#{1,6} [^ ].*$`)

// judgeForm reads one document and returns what is wrong with it, one line per
// finding, each naming the file, the line and the rule.
//
// Everything inside a fenced code block is exempt from every rule but the one
// that says the fence has to close. A fence holds bytes quoted from somewhere
// else, and a leg that normalised those would be rewriting the evidence a
// document is carrying.
func judgeForm(rel, body string) []string {
	var problems []string
	report := func(line int, what string) {
		problems = append(problems, fmt.Sprintf("%s:%d: %s", rel, line, what))
	}

	if body == "" {
		return []string{rel + ": the file is empty"}
	}
	if !strings.HasSuffix(body, "\n") {
		problems = append(problems, rel+": the last line has no newline after it")
	}
	if strings.HasSuffix(body, "\n\n") {
		problems = append(problems, rel+": the file ends with a blank line")
	}

	lines := strings.Split(strings.TrimSuffix(body, "\n"), "\n")
	fence := ""
	fenceOpenedAt := 0
	blanks := 0

	for i, line := range lines {
		n := i + 1

		// A fenced block is a run of lines this leg does not judge, and it also
		// interrupts the run of blank lines around it. Counting a blank line
		// before a block together with one after it would report a second blank
		// line in a row at a place where the file holds no such thing.
		if marker := fenceMarker(line); marker != "" {
			switch {
			case fence == "":
				fence, fenceOpenedAt = marker, n
			case strings.HasPrefix(marker, fence):
				fence = ""
			}
			blanks = 0
			continue
		}
		if fence != "" {
			blanks = 0
			continue
		}

		if strings.TrimSpace(line) == "" {
			blanks++
			if blanks == 2 {
				report(n, "a second blank line in a row; one separates, two say nothing the first did not")
			}
			continue
		}
		blanks = 0

		if strings.TrimRight(line, " \t") != line {
			report(n, "the line ends in whitespace, which is invisible in every editor that made it")
		}
		if strings.Contains(line, "\t") {
			report(n, "the line holds a tab, whose width is a setting rather than a property of the file")
		}
		if strings.HasPrefix(line, "#") && !heading.MatchString(line) {
			report(n, "the heading is not one to six hashes, one space and a title")
		}
		if heading.MatchString(line) && strings.HasSuffix(line, "#") {
			report(n, "the heading is closed with hashes, which the form does not use")
		}
	}

	if fence != "" {
		report(fenceOpenedAt, "the code fence opened here is never closed")
	}
	if strings.TrimSpace(lines[0]) == "" {
		problems = append(problems, rel+":1: the file starts with a blank line")
	}
	return problems
}

// fenceMarker returns the run of backticks or tildes a line opens or closes a
// fenced block with, and the empty string where the line is not a fence.
func fenceMarker(line string) string {
	trimmed := strings.TrimLeft(line, " ")
	for _, char := range []string{"```", "~~~"} {
		if strings.HasPrefix(trimmed, char) {
			run := 0
			for run < len(trimmed) && string(trimmed[run]) == string(char[0]) {
				run++
			}
			return trimmed[:run]
		}
	}
	return ""
}

// documentationLinksLeg refuses a link inside the Markdown that points at
// nothing in this tree.
//
// Only internal links are judged. An external one is somebody else's server,
// which is down for reasons that are not a reason to refuse a change here, and
// it is checked on a schedule instead by tools/externallinks.
//
// A fragment is not resolved. Nothing here reads a document's heading structure,
// so a link to a section that was renamed passes, and that is the bound on this
// leg rather than something it hides.
func documentationLinksLeg(root string) outcome {
	docs, err := markdownFiles(root)
	if err != nil {
		return fail(err.Error())
	}
	if len(docs) == 0 {
		return skip("there is no Markdown file in this tree, so no link was read")
	}

	var problems []string
	links := 0

	for _, rel := range docs {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return fail(err.Error())
		}
		for _, found := range internalLinks(rel, string(raw)) {
			links++
			if !existsInTree(root, rel, found.target) {
				problems = append(problems, fmt.Sprintf("%s:%d: the link to %s resolves to nothing in this tree", rel, found.line, found.target))
			}
		}
	}

	if len(problems) > 0 {
		return fail(strings.Join(problems, "\n") +
			"\n\nA link that resolves to nothing is a document describing a tree that is not" +
			"\nthis one. " + formDoc + " states what this leg reads and what it does not.")
	}
	if links == 0 {
		return skip(fmt.Sprintf("%d document(s) hold no internal link, so nothing was resolved", len(docs)))
	}
	return note(fmt.Sprintf("%d internal link(s) resolved, across %d document(s).", links, len(docs)))
}

// documentationPathsLeg refuses a path named in prose that is not in the tree.
//
// The subject is a path written in a backtick span, which is how every document
// here names a file, and the scope is every Markdown file in the tree including
// the ones at the repository root. A document imposing a rule is therefore
// inside the reach of the mechanism that reads it, which is the case a scope
// stopping at docs/ would have missed.
//
// A span is treated as a path when it holds a slash and nothing that says it is
// something else. Naming a path outside a backtick span is what a document does
// where the path is not meant to exist, and formDoc says so, because a rule with
// no way out is one somebody switches off.
func documentationPathsLeg(root string) outcome {
	docs, err := markdownFiles(root)
	if err != nil {
		return fail(err.Error())
	}
	if len(docs) == 0 {
		return skip("there is no Markdown file in this tree, so no path was read")
	}

	var problems []string
	named := 0

	for _, rel := range docs {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return fail(err.Error())
		}
		for _, found := range namedPaths(string(raw)) {
			if !aboutThisTree(root, found.target) {
				continue
			}
			named++
			if !existsInTree(root, "", found.target) {
				problems = append(problems, fmt.Sprintf("%s:%d: the document names %s, which is not in this tree", rel, found.line, found.target))
			}
		}
	}

	if len(problems) > 0 {
		return fail(strings.Join(problems, "\n") +
			"\n\nEither the path moved and the document did not, or the document means a path" +
			"\nit does not intend to resolve, in which case " + formDoc + " says to write it" +
			"\nwithout the backticks.")
	}
	if named == 0 {
		return skip(fmt.Sprintf("%d document(s) name no path, so nothing was resolved", len(docs)))
	}
	return note(fmt.Sprintf("%d path(s) named in prose resolved, across %d document(s).", named, len(docs)))
}

// reference is one thing a document points at, and the line it points from.
type reference struct {
	line   int
	target string
}

var (
	// inlineLink takes the target out of [text](target), with or without a
	// trailing title.
	inlineLink = regexp.MustCompile(`\[[^\]]*\]\(\s*([^)\s]+)(?:\s+"[^"]*")?\s*\)`)

	// referenceDefinition takes the target out of [label]: target.
	referenceDefinition = regexp.MustCompile(`^ {0,3}\[[^\]]+\]:\s*(\S+)`)

	// codeSpan matches a backtick span, which is how this tree's documents name
	// a file.
	codeSpan = regexp.MustCompile("`([^`\n]+)`")

	// scheme matches the front of a URL, so an external link and a mailto are
	// left to the schedule rather than resolved against the tree.
	scheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)
)

// internalLinks returns every link in a document that points inside this tree.
// Links inside a fenced block are quoted rather than written and are left alone.
func internalLinks(rel, body string) []reference {
	var out []reference
	forEachProseLine(body, func(n int, line string) {
		for _, m := range inlineLink.FindAllStringSubmatch(withoutCodeSpans(line), -1) {
			if target, ok := internalTarget(m[1]); ok {
				out = append(out, reference{line: n, target: target})
			}
		}
		if m := referenceDefinition.FindStringSubmatch(line); m != nil {
			if target, ok := internalTarget(m[1]); ok {
				out = append(out, reference{line: n, target: target})
			}
		}
	})
	return out
}

// internalTarget says whether a link target is one this tree can resolve, and
// what is left of it once the fragment is off.
func internalTarget(target string) (string, bool) {
	target = strings.Trim(target, "<>")
	if target == "" || strings.HasPrefix(target, "#") || scheme.MatchString(target) {
		return "", false
	}
	if i := strings.IndexByte(target, '#'); i >= 0 {
		target = target[:i]
	}
	if target == "" {
		return "", false
	}
	return target, true
}

// namedPaths returns every path into this tree that a document names, wherever
// it names it.
//
// The subject is the whole of the prose and not only what is inside link syntax
// or inside a backtick span. Every document here names files both ways, often in
// the same sentence, and a check reading only the marked-up half would pass a
// document whose plain sentences describe a tree that no longer exists.
//
// A token is what falls out of splitting a line on whitespace, with the
// punctuation a sentence puts around a path taken off both ends. Whether what is
// left is a path at all is treePath's decision, and whether it is a path into
// this tree is aboutThisTree's.
func namedPaths(body string) []reference {
	var out []reference
	forEachProseLine(body, func(n int, line string) {
		seen := map[string]bool{}
		for _, token := range strings.Fields(line) {
			// Trimmed at each end with its own set. A dot leads a path here,
			// in .github, and ends a sentence that named one, so a single
			// cutset would either take the leading dot off a real path or
			// leave a full stop on the end of one.
			token = strings.TrimLeft(token, "`\"'([{<*_")
			target, ok := treePath(strings.TrimRight(token, "`\"')]}>,;:.!?*_"))
			if !ok || seen[target] {
				continue
			}
			seen[target] = true
			out = append(out, reference{line: n, target: target})
		}
	})
	return out
}

// treePath decides whether a backtick span names a path in this repository.
//
// The rule errs towards reading less. A span with no slash is a bare name, and
// this tree's documents use those for things that are not files at all. A span
// carrying an @, a colon or a first segment with a dot inside it is a module
// path, a URL or a pinned version, none of which resolve against a checkout. A
// span carrying a shell metacharacter is a command. What is left is a relative
// path somebody wrote meaning this tree, and the cost of the rule is that a file
// named without a slash is not checked at all.
func treePath(span string) (string, bool) {
	span = strings.TrimSpace(span)
	if span == "" || strings.ContainsAny(span, " \t") || !strings.Contains(span, "/") {
		return "", false
	}
	if strings.ContainsAny(span, "@:*?$<>|()=,'\"\\") {
		return "", false
	}
	if strings.HasSuffix(span, "...") {
		return "", false
	}
	span = strings.TrimPrefix(span, "./")
	if strings.HasPrefix(span, "/") || strings.HasPrefix(span, "..") {
		return "", false
	}
	first, _, _ := strings.Cut(span, "/")
	if strings.Contains(strings.TrimPrefix(first, "."), ".") {
		return "", false
	}
	return strings.TrimSuffix(span, "/"), true
}

// aboutThisTree says whether a span that looks like a path is a path into this
// repository, by asking whether its first segment is an entry at the root.
//
// This is what keeps the leg off the things a document about this project has to
// name and which are not files here: a standard library import path, a GOOS and
// GOARCH pair, the layout inside a bundle a run writes. All three are two words
// with a slash between them and no reading of the string separates them from a
// relative path.
//
// What it costs is stated rather than hidden. A path under a top level directory
// that does not exist is not read at all, so renaming a top level directory
// stops this leg reading every path under the old name instead of refusing them.
// The links leg still resolves whatever is written as a link, and that is the
// half of the cover this bound does not remove.
func aboutThisTree(root, target string) bool {
	first, _, _ := strings.Cut(target, "/")
	if first == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(first)))
	return err == nil
}

// existsInTree resolves a target and says whether it is there. A target
// beginning with a slash is read against the repository root; anything else is
// read against the directory of the document naming it, and against the root
// where from is empty.
func existsInTree(root, from, target string) bool {
	var rel string
	switch {
	case strings.HasPrefix(target, "/"):
		rel = strings.TrimPrefix(target, "/")
	case from == "":
		rel = target
	default:
		rel = path.Join(path.Dir(from), target)
	}
	rel = path.Clean(rel)
	if rel == "." || strings.HasPrefix(rel, "..") {
		return false
	}
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

// forEachProseLine calls back for every line outside a fenced code block, with
// CRLF already normalised so a checkout cannot change what is read.
func forEachProseLine(body string, do func(n int, line string)) {
	fence := ""
	for i, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		if marker := fenceMarker(line); marker != "" {
			switch {
			case fence == "":
				fence = marker
			case strings.HasPrefix(marker, fence):
				fence = ""
			}
			continue
		}
		if fence == "" {
			do(i+1, line)
		}
	}
}

// withoutCodeSpans blanks the backtick spans in a line, so a link written inside
// one is read as the quotation it is rather than as a link.
func withoutCodeSpans(line string) string {
	return codeSpan.ReplaceAllStringFunc(line, func(s string) string {
		return strings.Repeat(" ", len(s))
	})
}

// markdownFiles lists every Markdown file in the tree, as slash separated paths
// relative to the root, in walk order. The repository root is included, which is
// the scope condition the path leg's comment names.
func markdownFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(p), ".md") {
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out, err
}
