// Package version answers one question, what to call this build, and is
// required never to answer it with an empty string.
//
// Nothing here reads the clock, the build host or an absolute path, so two
// builds of one commit agree on what they print. That is a property the
// reproducible build work depends on and it is cheaper to hold from the start
// than to recover later.
package version

import (
	"runtime/debug"
	"strings"
)

// Set at link time by a release build:
//
//	go build -ldflags "-X github.com/iderex/gegenprobe/internal/version.tag=v0.1.0"
//
// Neither is required. When tag is empty the revision is used, and when the
// build carries no revision either, Describe says so rather than inventing one.
var (
	tag    string
	commit string
)

// Unknown is what a build with no repository metadata at all says about itself.
// It is a sentence rather than an empty string or a zero version because a
// clone unpacked from a tarball genuinely does not know, and a build claiming a
// version it cannot know is the failure this constant exists to prevent.
const Unknown = "unknown, built without repository metadata"

// shortCommit is how many hex digits of a revision are printed. Twelve is long
// enough to be unambiguous in a repository of this size and short enough to
// read back over a phone.
const shortCommit = 12

// Stamp is everything a build knows about where it came from.
type Stamp struct {
	Tag      string
	Commit   string
	Modified bool
}

// Describe returns the string the version subcommand prints. It is never empty.
func Describe() string { return describe(current()) }

// current merges what the linker was told with what the toolchain recorded.
// A value passed at link time wins, because a release build states its tag
// deliberately and the toolchain cannot know it.
func current() Stamp {
	s := Stamp{Tag: tag, Commit: commit}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return s
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if s.Commit == "" {
				s.Commit = setting.Value
			}
		case "vcs.modified":
			s.Modified = setting.Value == "true"
		}
	}
	return s
}

func describe(s Stamp) string {
	var b strings.Builder
	switch {
	case s.Tag != "":
		b.WriteString(s.Tag)
	case s.Commit != "":
		b.WriteString("untagged, built from commit ")
		b.WriteString(short(s.Commit))
	default:
		// No tag and no revision. Whether the tree was modified is not
		// knowable either, so nothing is appended to this.
		return Unknown
	}
	if s.Modified {
		b.WriteString(", with uncommitted changes")
	}
	return b.String()
}

func short(commit string) string {
	if len(commit) <= shortCommit {
		return commit
	}
	return commit[:shortCommit]
}
