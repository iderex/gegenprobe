package commit

import (
	"fmt"
	"io"
)

// Report writes what the judgement found and what it covered. The count of
// commits examined is printed whether or not anything was found, because a run
// over an empty range and a run that found nothing are different statements and
// only one of them is evidence.
func Report(w io.Writer, commits []Commit, findings []Finding, origin Origin) {
	for _, f := range findings {
		mark := "refused"
		if !f.Fatal {
			mark = "reported"
		}
		fmt.Fprintf(w, "%s: %s\n", mark, f)
	}

	refusing := 0
	for _, f := range findings {
		if f.Fatal {
			refusing++
		}
	}

	fmt.Fprintf(w, "%d commit(s) examined, %d finding(s), %d of them refusing.\n",
		len(commits), len(findings), refusing)

	if origin == External && refusing < len(findings) {
		fmt.Fprintln(w, "The head of this change is not a branch of this repository, so a subject "+
			"naming no issue is reported rather than refused. The linkage is added when the "+
			"contribution is handled.")
	}
}
