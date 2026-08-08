package version

import (
	"runtime/debug"
	"strings"
	"testing"
)

func buildInfo(settings ...debug.BuildSetting) *debug.BuildInfo {
	return &debug.BuildInfo{Settings: settings}
}

func TestFromBuildInfoShortensACleanRevision(t *testing.T) {
	got := fromBuildInfo(buildInfo(
		debug.BuildSetting{Key: "vcs.revision", Value: "9163b4d36a5906125addb3220a1a95845f24be9a"},
		debug.BuildSetting{Key: "vcs.modified", Value: "false"},
	))
	if want := "9163b4d36a59"; got != want {
		t.Fatalf("version of a clean checkout = %q, want %q", got, want)
	}
}

func TestFromBuildInfoSaysWhenTheTreeWasModified(t *testing.T) {
	got := fromBuildInfo(buildInfo(
		debug.BuildSetting{Key: "vcs.revision", Value: "9163b4d36a5906125addb3220a1a95845f24be9a"},
		debug.BuildSetting{Key: "vcs.modified", Value: "true"},
	))
	if !strings.HasPrefix(got, "9163b4d36a59") {
		t.Errorf("version of a modified checkout = %q, want it to start with the short revision", got)
	}
	if !strings.Contains(got, "modified") {
		t.Errorf("version of a modified checkout = %q, want it to say the tree was modified", got)
	}
}

// A build from an unpacked tarball reaches this shape: build information
// exists, and it carries no vcs keys at all. The string it produces is the one
// thing an operator will paste into a report, so it has to say that it does not
// know rather than be empty.
func TestFromBuildInfoWithoutGitMetadataSaysSoAndIsNotEmpty(t *testing.T) {
	got := fromBuildInfo(buildInfo())
	if got == "" {
		t.Fatal("version without git metadata is empty, which is the one answer that cannot be read")
	}
	if !strings.Contains(got, "unknown") {
		t.Errorf("version without git metadata = %q, want it to say it is unknown", got)
	}
}

// Resolve prefers the tag over anything the toolchain recorded, because a
// release build is the only build that knows which release it is.
func TestResolvePrefersTheStampedTag(t *testing.T) {
	old := stamped
	t.Cleanup(func() { stamped = old })

	stamped = "  v0.1.0  "
	if got, want := Resolve(), "v0.1.0"; got != want {
		t.Fatalf("Resolve with a stamped tag = %q, want %q", got, want)
	}
}

// Whitespace is what a build system supplies when a shell substitution produced
// nothing, and it must not be mistaken for a tag.
func TestResolveTreatsAWhitespaceStampAsAbsent(t *testing.T) {
	old := stamped
	t.Cleanup(func() { stamped = old })

	stamped = "   "
	if got := Resolve(); got == "" || strings.TrimSpace(got) == "" {
		t.Fatalf("Resolve with a whitespace stamp = %q, want it to fall back to the build information", got)
	}
}

// Resolve is called twice in one process here, which is the cheapest statement
// of the property #28 proves over the binary: nothing in it reads the clock.
func TestResolveIsStable(t *testing.T) {
	if first, second := Resolve(), Resolve(); first != second {
		t.Fatalf("Resolve returned %q then %q; the version must not depend on when it was asked", first, second)
	}
}
