// Package version answers one question: which build of this program is
// running.
//
// A version string that is guessed reads exactly like one that is known, and
// the place that difference matters is a bug report from an operator whose
// numbers disagree with somebody else's. So every answer here says where it
// came from, and the answer for a build that cannot know is a sentence saying
// so rather than an empty string or a zero.
//
// Three sources, in this order. A release build passes the tag in through the
// linker. A build from a checkout takes the commit from the build information
// the toolchain records. A build from an unpacked tarball has neither, and gets
// a sentence.
//
// Nothing here reads the clock, the build host or the build directory, so two
// builds of one commit produce one string. That is a precondition of the
// reproducibility work in #28 rather than a nicety, and #28 is what proves it
// over the binary rather than over this function.
package version

import (
	"runtime/debug"
	"strings"
)

// stamped is empty in every build except a release, which sets it to the tag:
//
//	go build -ldflags "-X github.com/iderex/gegenprobe/internal/version.stamped=v0.1.0"
//
// It is a variable rather than a constant because the linker can only write to
// a variable, and it is unexported so that the only way to set it is that flag.
var stamped string

// shortRevision is how much of a commit sha the version string carries. Twelve
// is long enough to be unambiguous in a repository of this size and short
// enough to be read out over a call.
const shortRevision = 12

// Resolve returns the version string this build should print.
func Resolve() string {
	if tag := strings.TrimSpace(stamped); tag != "" {
		return tag
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown, this build carries no build information"
	}
	return fromBuildInfo(info)
}

// fromBuildInfo derives the version from what the toolchain recorded about the
// checkout it built from. It is separate from Resolve so the three shapes it
// has to handle can be tested without arranging three builds.
func fromBuildInfo(info *debug.BuildInfo) string {
	var revision, modified string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value
		}
	}

	if revision == "" {
		return "unknown, this build has no git metadata to take a version from"
	}
	if len(revision) > shortRevision {
		revision = revision[:shortRevision]
	}
	if modified == "true" {
		return revision + ", built from a modified working tree"
	}
	return revision
}
