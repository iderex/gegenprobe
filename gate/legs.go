package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// legs is the list, in the order it runs. A check added by a later issue is
// added here and nowhere else, which is what keeps one entry point from becoming
// two.
//
// The order is the language checks first, cheapest of them first, and then this
// repository's own checks. A run that stops early has usually stopped on the
// thing that takes a second to fix.
func legs() []leg {
	return []leg{
		{
			name:    "format",
			subject: "every Go file in the tree is gofmt formatted",
			run:     formatLeg,
		},
		{
			name:    "vet",
			subject: "go vet ./...",
			run:     func(root string) outcome { return command(root, "go", "vet", "./...") },
		},
		{
			name:    "static analysis",
			subject: "staticcheck ./...",
			run:     staticcheckLeg,
		},
		{
			name:    "tests",
			subject: "go test ./...",
			run:     func(root string) outcome { return command(root, "go", "test", "./...") },
		},
		{
			name:    "decision records",
			subject: "every record under docs/decisions follows the format 0000 fixes",
			run:     decisionRecordsLeg,
		},
		{
			name:    "decision index",
			subject: "docs/decisions/README.md is derivable from the records beside it",
			run:     func(root string) outcome { return command(root, "go", "run", "./tools/decisionindex", "-check") },
		},
		{
			name:    "action pinning",
			subject: "every action a workflow uses is pinned to a commit sha",
			run:     actionPinningLeg,
		},
		{
			name:    "fixture storage",
			subject: "every fixture is stored so that no checkout can rewrite its bytes, and says where it came from",
			run:     fixtureStorageLeg,
		},
		{
			name:    "reproducible build",
			subject: "two builds of this tree produce identical bytes, and neither carries the directory they were built in",
			run:     reproducibleBuildLeg,
		},
	}
}

// formatLeg judges the formatting of every Go file in the tree without shelling
// out to gofmt, so that the verdict is the same wherever it runs.
//
// It compares after normalising CRLF to LF. A checkout on Windows can hold every
// file with carriage returns, which gofmt reports as a formatting difference on
// every line of every file and which says nothing about how the source is
// written. What the bytes in a checkout are is a separate question, held by #23
// and #24, and this leg says when it normalised rather than passing in silence.
func formatLeg(root string) outcome {
	var unformatted []string
	carriageReturns := 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(src, []byte("\r\n")) {
			carriageReturns++
			src = bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))
		}
		want, err := format.Source(src)
		if err != nil {
			unformatted = append(unformatted, filepath.ToSlash(path)+": "+err.Error())
			return nil
		}
		if !bytes.Equal(src, want) {
			unformatted = append(unformatted, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		return fail(err.Error())
	}

	if len(unformatted) > 0 {
		return fail(strings.Join(unformatted, "\n") + "\n\nrun: gofmt -w " + strings.Join(shortest(unformatted), " "))
	}
	if carriageReturns > 0 {
		return note(fmt.Sprintf("%d file(s) hold CRLF line endings in this checkout and were judged after normalising to LF. Line endings are not this leg's subject.", carriageReturns))
	}
	return pass()
}

// staticcheckLeg runs staticcheck where it is installed. Where it is not, the
// leg is skipped and says so with the command that would install it, because a
// static analysis leg that quietly passes on a machine without the analyser is
// the exact shape this command exists to refuse.
func staticcheckLeg(root string) outcome {
	if _, err := exec.LookPath("staticcheck"); err != nil {
		return skip("staticcheck is not on PATH, so nothing on this run performed static analysis. " +
			"Install it with: go install honnef.co/go/tools/cmd/staticcheck@2025.1.1")
	}
	return command(root, "staticcheck", "./...")
}

// command runs one program at the root and turns what it wrote into the leg's
// detail, so a failure carries the tool's own words rather than an exit status a
// reader would have to reproduce.
func command(root, name string, args ...string) outcome {
	cmd := exec.Command(name, args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		return pass()
	}
	detail := strings.TrimRight(string(out), "\r\n")
	if detail == "" {
		detail = "(no output)"
	}
	return fail(detail + "\n\n" + name + " " + strings.Join(args, " ") + ": " + err.Error())
}

// shortest keeps the repair line usable when many files are listed: a parse
// error carries a colon and its message, and only the path is useful in a
// command.
func shortest(entries []string) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, strings.SplitN(e, ":", 2)[0])
	}
	return out
}
