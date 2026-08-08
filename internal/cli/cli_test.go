package cli

import (
	"bytes"
	"strings"
	"testing"
)

func invoke(t *testing.T, args ...string) (status int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	status = Run(args, &out, &errOut)
	return status, out.String(), errOut.String()
}

func TestVersionPrintsAVersionOnStdoutAndSucceeds(t *testing.T) {
	status, stdout, stderr := invoke(t, "version")
	if status != 0 {
		t.Errorf("status = %d, want 0", status)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Error("version printed nothing on stdout")
	}
	if stderr != "" {
		t.Errorf("version wrote %q on stderr, want nothing", stderr)
	}
}

// The exit status is the half of this a script reads, so it is asserted
// separately from the text: a usage message on a successful exit is a command
// that did nothing and said so quietly.
func TestNoArgumentsPrintsUsageAndFails(t *testing.T) {
	status, stdout, stderr := invoke(t)
	if status == 0 {
		t.Error("status = 0 for an invocation with no subcommand, want non zero")
	}
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("stderr = %q, want it to carry the usage text", stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want the usage of a failed invocation to go to stderr", stdout)
	}
}

func TestHelpPrintsUsageOnStdoutAndSucceeds(t *testing.T) {
	status, stdout, stderr := invoke(t, "help")
	if status != 0 {
		t.Errorf("status = %d, want 0", status)
	}
	if !strings.Contains(stdout, "usage:") {
		t.Errorf("stdout = %q, want it to carry the usage text", stdout)
	}
	if stderr != "" {
		t.Errorf("help wrote %q on stderr, want nothing", stderr)
	}
}

func TestUnknownSubcommandNamesItAndFails(t *testing.T) {
	status, _, stderr := invoke(t, "compare")
	if status == 0 {
		t.Error("status = 0 for an unknown subcommand, want non zero")
	}
	if !strings.Contains(stderr, `"compare"`) {
		t.Errorf("stderr = %q, want it to name the subcommand that was not understood", stderr)
	}
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("stderr = %q, want it to carry the usage text", stderr)
	}
}

// A trailing argument is the shape of somebody expecting a flag this binary
// does not have. Refusing it is what keeps a later `version --json` from having
// silently meant `version` for a release.
func TestSubcommandsRefuseTrailingArguments(t *testing.T) {
	for _, sub := range []string{"version", "help"} {
		status, _, stderr := invoke(t, sub, "--json")
		if status == 0 {
			t.Errorf("%s --json: status = 0, want non zero", sub)
		}
		if !strings.Contains(stderr, "--json") {
			t.Errorf("%s --json: stderr = %q, want it to quote the argument it refused", sub, stderr)
		}
	}
}

// The usage text is what somebody reads before they know anything else, so it
// names every subcommand the switch above accepts. A subcommand added without a
// line here is a feature only its author can find.
func TestUsageNamesEverySubcommand(t *testing.T) {
	for _, sub := range []string{"version", "help"} {
		if !strings.Contains(usage, "\n    "+sub+" ") {
			t.Errorf("usage does not list the %q subcommand", sub)
		}
	}
}
