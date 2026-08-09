package main

import (
	"bytes"
	"strings"

	"github.com/iderex/gegenprobe/internal/commit"
)

// A commit message is the one artefact in this repository that cannot be
// corrected. A bad line in a file is repaired by a later commit; a bad line in a
// message is repaired only by rewriting history, which on a branch somebody else
// has pulled is not a repair at all. So the two rules over messages are worth
// meeting before a push rather than after one, and this leg is where a
// contributor meets them.
//
// The judgement is not here. It is internal/commit, which tools/commithygiene
// also calls, so the verdict before a push and the verdict the job named Commit
// hygiene reaches on a pull request come from one place rather than from two
// copies that drift. What differs is only the range: this leg judges the work
// between the default branch and HEAD, and the job judges the range the pull
// request actually holds, which is the only place the base and the head are
// known.
//
// What this leg can see is bounded by the clone it runs in. Where the default
// branch is not fetched, or where the work is on it and the range is empty,
// there is nothing to judge and the leg reports that it examined nothing rather
// than passing. A shallow checkout is the ordinary shape on a runner, so that
// skip is expected there and is evidence of nothing.
const commitDeclaration = "docs/commit-messages.md"

func commitHygieneLeg(root string) outcome {
	var out bytes.Buffer

	result, err := commit.Run(&out, root, commitDeclaration, "origin/main", "HEAD", commit.Internal)
	if err != nil {
		return fail(err.Error())
	}
	if result.NoRange != "" {
		return skip(result.NoRange + ", so no commit was judged.\n\n" +
			"Fetch the default branch, or judge an explicit range: " +
			"go run ./tools/commithygiene -base <ref> -head <ref>")
	}

	detail := strings.TrimRight(out.String(), "\n")
	if commit.Failed(result.Findings) {
		return fail(detail + "\n\nThe rules are in " + commitDeclaration + ", and widening one is an edit there.")
	}
	return note(detail)
}
