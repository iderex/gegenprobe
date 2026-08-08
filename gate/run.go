package main

import (
	"fmt"
	"io"
	"strings"
)

// verdict is what a leg reports about itself. A leg that could not run reports
// skipped rather than passed, because a check that examined nothing and a check
// that found nothing are different statements and only one of them is evidence.
type verdict int

const (
	passed verdict = iota
	failed
	skipped
)

func (v verdict) String() string {
	switch v {
	case passed:
		return "pass"
	case failed:
		return "FAIL"
	default:
		return "skip"
	}
}

// outcome carries the verdict and the words that go with it. For a failure the
// detail is what the check actually said, so a reader does not have to rerun it
// to find out what was wrong. For a skip it is what was missing and what
// obtaining it would take.
type outcome struct {
	verdict verdict
	detail  string
}

func pass() outcome               { return outcome{verdict: passed} }
func fail(detail string) outcome  { return outcome{verdict: failed, detail: detail} }
func skip(missing string) outcome { return outcome{verdict: skipped, detail: missing} }
func note(d string) outcome       { return outcome{verdict: passed, detail: d} }

// leg is one check. The subject says what a pass covers, printed beside the
// name, so that a green line is read as a statement about something specific
// rather than as general reassurance.
type leg struct {
	name    string
	subject string
	run     func(root string) outcome
}

// run executes the legs in the order given and stops at the first failure
// without running the rest. It returns the exit status: zero where every leg
// that ran passed, one otherwise.
func run(w io.Writer, root string, ls []leg) int {
	fmt.Fprintf(w, "gate: %d legs, in order: %s\n\n", len(ls), strings.Join(names(ls), ", "))

	width := 0
	for _, l := range ls {
		if len(l.name) > width {
			width = len(l.name)
		}
	}

	var skippedLegs []string
	passedCount := 0

	for i, l := range ls {
		o := l.run(root)
		fmt.Fprintf(w, "%-*s  %s  %s\n", width, l.name, o.verdict, l.subject)

		switch o.verdict {
		case passed:
			passedCount++
			if o.detail != "" {
				fmt.Fprintf(w, "%s\n", indent(o.detail))
			}
		case skipped:
			skippedLegs = append(skippedLegs, l.name)
			fmt.Fprintf(w, "%s\n", indent("not run: "+o.detail))
		case failed:
			fmt.Fprintf(w, "%s\n", indent(o.detail))
			fmt.Fprintf(w, "\ngate: FAILED at leg %d of %d, %s.\n", i+1, len(ls), l.name)
			if rest := names(ls[i+1:]); len(rest) > 0 {
				fmt.Fprintf(w, "gate: %d leg(s) did not run: %s.\n", len(rest), strings.Join(rest, ", "))
			} else {
				fmt.Fprintln(w, "gate: it was the last leg, so every other leg ran.")
			}
			if len(skippedLegs) > 0 {
				fmt.Fprintf(w, "gate: %d leg(s) examined nothing: %s.\n", len(skippedLegs), strings.Join(skippedLegs, ", "))
			}
			return 1
		}
	}

	fmt.Fprintf(w, "\ngate: %d leg(s), %d passed, %d skipped, 0 failed.\n", len(ls), passedCount, len(skippedLegs))
	if len(skippedLegs) > 0 {
		fmt.Fprintf(w, "gate: %d leg(s) examined nothing: %s. This run covered less than the whole set.\n",
			len(skippedLegs), strings.Join(skippedLegs, ", "))
	}
	return 0
}

func names(ls []leg) []string {
	out := make([]string, 0, len(ls))
	for _, l := range ls {
		out = append(out, l.name)
	}
	return out
}

// indent puts every line of a detail under the leg it belongs to, including the
// lines a tool wrote itself, so that a multi line failure cannot be mistaken for
// output from the leg after it.
func indent(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = "    " + l
	}
	return strings.Join(lines, "\n")
}
