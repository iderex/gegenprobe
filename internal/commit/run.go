package commit

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Format is what git is asked to print, and it is here rather than at the call
// site so that the bytes the fixtures hold are the bytes the judgement receives.
// The two separators are the unit separator between the fields and, from git's
// own -z, a null byte between the commits. Both are outside the allowlist this
// package judges against, so neither can appear inside a message that passed and
// neither can appear inside a sha, an author or a parent list at all.
const Format = "%H%x1f%an <%ae>%x1f%P%x1f%B"

// Result is what one judgement covered and what it found. NoRange is non empty
// where there was nothing to judge, which is a different statement from finding
// nothing and is never reported as a pass.
type Result struct {
	Commits  []Commit
	Findings []Finding
	NoRange  string
}

// Run reads the declaration, resolves the range and judges it, writing the
// report as it goes. It is the whole of what the leg named commit hygiene does
// and the whole of what tools/commithygiene does, so the verdict a contributor
// meets before pushing and the verdict a pull request meets afterwards come from
// one place rather than from two that drift.
func Run(w io.Writer, dir, doc, base, head string, origin Origin) (Result, error) {
	declaration, err := os.ReadFile(filepath.Join(dir, doc))
	if err != nil {
		return Result{}, err
	}
	rules, err := ParseRules(string(declaration))
	if err != nil {
		return Result{}, fmt.Errorf("%s: %v", doc, err)
	}

	for _, ref := range []string{base, head} {
		out, err := git(dir, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
		if err != nil || strings.TrimSpace(out) == "" {
			return Result{NoRange: fmt.Sprintf("%s names no commit in this clone", ref)}, nil
		}
	}

	stream, err := git(dir, "log", "-z", "--format="+Format, base+".."+head)
	if err != nil {
		return Result{}, err
	}
	commits, err := Parse([]byte(stream))
	if err != nil {
		return Result{}, err
	}
	if len(commits) == 0 {
		return Result{NoRange: fmt.Sprintf("%s..%s holds no commit", base, head)}, nil
	}

	findings := Judge(commits, rules, origin)
	Report(w, commits, findings, origin)
	return Result{Commits: commits, Findings: findings}, nil
}

// git runs one read only git command and hands back what it printed. Nothing
// here writes to the repository, so a failure is a failure to read history
// rather than a change left half made.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		detail := ""
		if ee, ok := err.(*exec.ExitError); ok {
			detail = strings.TrimSpace(string(ee.Stderr))
		}
		if detail == "" {
			return "", fmt.Errorf("git %s: %v", strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, detail)
	}
	return string(out), nil
}
