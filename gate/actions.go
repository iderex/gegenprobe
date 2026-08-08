package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// A tag is mutable. An action referenced by one runs whatever its author last
// pushed under that name, inside a job holding a token, so the reference is
// pinned to a commit sha and this leg is what keeps it pinned. The audit in
// .github/workflows/zizmor.yml reports the same thing on the server; this
// refuses it before a push.
//
// The leg reads the workflow files as text rather than as YAML. What it judges
// is one field on one line, the file that holds it is the thing a reader edits,
// and a parser would add a dependency this tree does not carry for no property
// it does not already have. The cost is that a `uses:` written as a folded or
// quoted scalar across two lines would not be seen, which no workflow in this
// tree does and which the failure message says nothing to hide.
var (
	// pinnedRef is a commit sha and nothing else: forty lowercase hexadecimal
	// digits. A branch, a tag and a short sha all fail it.
	pinnedRef = regexp.MustCompile(`^[0-9a-f]{40}$`)

	// usesLine takes the reference off a step's uses field, with or without the
	// leading list dash.
	usesLine = regexp.MustCompile(`^\s*(?:-\s+)?uses:\s*(\S+)`)
)

// actionPinningLeg refuses an action reference in .github/workflows that is not
// a full commit sha.
func actionPinningLeg(root string) outcome {
	dir := filepath.Join(root, ".github", "workflows")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return skip("there is no .github/workflows directory in this tree, so no action reference was read")
	}
	if err != nil {
		return fail(err.Error())
	}

	var refused []string
	read := 0

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if ext := filepath.Ext(e.Name()); ext != ".yml" && ext != ".yaml" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			return fail(err.Error())
		}
		read++
		for i, line := range strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				continue
			}
			m := usesLine.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			if why := unpinned(strings.Trim(m[1], `"'`)); why != "" {
				refused = append(refused, fmt.Sprintf("%s:%d: %s", filepath.ToSlash(path), i+1, why))
			}
		}
	}

	if len(refused) > 0 {
		return fail(strings.Join(refused, "\n") +
			"\n\nPin each one to the commit sha the reference resolves to today, and keep the" +
			"\nversion in a trailing comment so the file stays legible:" +
			"\n\n    gh api repos/OWNER/REPO/commits/TAG --jq .sha")
	}
	if read == 0 {
		return skip("there is no workflow file in .github/workflows, so no action reference was read")
	}
	return pass()
}

// unpinned says why a reference is not pinned, or returns the empty string where
// it is. The three admissible shapes are a commit sha, a container image digest
// and a path into this repository, which carries no ref to pin at all.
func unpinned(ref string) string {
	switch {
	case strings.HasPrefix(ref, "./"), ref == ".":
		return ""
	case strings.HasPrefix(ref, "docker://"):
		if _, digest, ok := strings.Cut(ref, "@sha256:"); ok && digest != "" {
			return ""
		}
		return ref + " names a container image by tag; use docker://IMAGE@sha256:DIGEST"
	}

	at := strings.LastIndex(ref, "@")
	if at < 0 {
		return ref + " carries no ref at all, so it runs whatever the default branch holds"
	}
	if got := ref[at+1:]; !pinnedRef.MatchString(got) {
		return ref + " is pinned to " + got + ", which is not a 40 character commit sha"
	}
	return ""
}
