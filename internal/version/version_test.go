package version

import (
	"strings"
	"testing"
)

func TestDescribe(t *testing.T) {
	for _, tc := range []struct {
		name  string
		stamp Stamp
		want  string
	}{
		{
			name:  "a tag wins, because a release build states it deliberately",
			stamp: Stamp{Tag: "v0.1.0", Commit: "0123456789abcdef0123456789abcdef01234567"},
			want:  "v0.1.0",
		},
		{
			name:  "no tag falls back to the revision, shortened",
			stamp: Stamp{Commit: "0123456789abcdef0123456789abcdef01234567"},
			want:  "untagged, built from commit 0123456789ab",
		},
		{
			name:  "a revision shorter than the cut is printed whole",
			stamp: Stamp{Commit: "0123456"},
			want:  "untagged, built from commit 0123456",
		},
		{
			name:  "a modified tree says so beside the tag",
			stamp: Stamp{Tag: "v0.1.0", Modified: true},
			want:  "v0.1.0, with uncommitted changes",
		},
		{
			name:  "a modified tree says so beside the revision",
			stamp: Stamp{Commit: "0123456789abcdef", Modified: true},
			want:  "untagged, built from commit 0123456789ab, with uncommitted changes",
		},
		{
			name:  "neither a tag nor a revision says exactly that",
			stamp: Stamp{},
			want:  Unknown,
		},
		{
			// A build with no revision cannot know whether the tree was
			// modified, so the flag is not carried into a sentence that
			// would imply it did know.
			name:  "no revision and a modified flag still says only that it does not know",
			stamp: Stamp{Modified: true},
			want:  Unknown,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := describe(tc.stamp); got != tc.want {
				t.Errorf("describe(%+v) = %q, want %q", tc.stamp, got, tc.want)
			}
		})
	}
}

// The empty string is the one answer this package may never give: the version
// subcommand printing a blank line is indistinguishable from a version that is
// genuinely blank, and a reader cannot tell a broken build from an old one.
func TestDescribeIsNeverEmpty(t *testing.T) {
	for _, s := range []Stamp{
		{},
		{Modified: true},
		{Tag: "v0.1.0"},
		{Commit: "0123456789abcdef"},
	} {
		if got := strings.TrimSpace(describe(s)); got == "" {
			t.Errorf("describe(%+v) returned nothing", s)
		}
	}
	if got := strings.TrimSpace(Describe()); got == "" {
		t.Error("Describe() returned nothing")
	}
}

// Describe reads no clock and no environment, so two calls in one process agree.
// The stronger property, that two builds of one commit agree, needs two builds
// and is proven in the root package.
func TestDescribeIsStable(t *testing.T) {
	first, second := Describe(), Describe()
	if first != second {
		t.Errorf("Describe() gave %q then %q", first, second)
	}
}
