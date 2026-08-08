package cli

import (
	"bytes"
	"strings"
	"testing"
)

func run(args ...string) (stdout, stderr string, status int) {
	var out, errOut bytes.Buffer
	status = Run(args, &out, &errOut)
	return out.String(), errOut.String(), status
}

func TestNoSubcommandPrintsUsageAndFails(t *testing.T) {
	stdout, stderr, status := run()
	if status == 0 {
		t.Error("running with no arguments succeeded; it has to fail")
	}
	if !strings.Contains(stderr, "usage: "+Name) {
		t.Errorf("usage did not reach stderr, got %q", stderr)
	}
	if stdout != "" {
		t.Errorf("a misuse wrote to stdout: %q", stdout)
	}
}

func TestVersionPrintsOneLineOnStdout(t *testing.T) {
	stdout, stderr, status := run("version")
	if status != 0 {
		t.Errorf("version exited %d", status)
	}
	if stderr != "" {
		t.Errorf("version wrote to stderr: %q", stderr)
	}
	line := strings.TrimSuffix(stdout, "\n")
	if line == "" {
		t.Error("version printed nothing")
	}
	if strings.Contains(line, "\n") {
		t.Errorf("version printed more than one line: %q", stdout)
	}
}

func TestHelpPrintsUsageOnStdoutAndSucceeds(t *testing.T) {
	stdout, stderr, status := run("help")
	if status != 0 {
		t.Errorf("help exited %d", status)
	}
	if stderr != "" {
		t.Errorf("help wrote to stderr: %q", stderr)
	}
	if !strings.Contains(stdout, "usage: "+Name) {
		t.Errorf("help printed no usage, got %q", stdout)
	}
}

func TestUnknownSubcommandFailsAndNamesIt(t *testing.T) {
	stdout, stderr, status := run("compare")
	if status == 0 {
		t.Error("an unknown subcommand succeeded")
	}
	if !strings.Contains(stderr, "compare") {
		t.Errorf("the diagnostic does not name the subcommand that was given: %q", stderr)
	}
	if stdout != "" {
		t.Errorf("a misuse wrote to stdout: %q", stdout)
	}
}

// A subcommand that quietly ignores an argument is a subcommand that will
// quietly ignore a filename somebody meant it to read.
func TestASubcommandRefusesArgumentsItDoesNotTake(t *testing.T) {
	for _, c := range commands() {
		stdout, stderr, status := run(c.name, "extra")
		if status == 0 {
			t.Errorf("%s accepted an argument it does not take", c.name)
		}
		if !strings.Contains(stderr, c.name) {
			t.Errorf("%s: the diagnostic does not name the subcommand: %q", c.name, stderr)
		}
		if stdout != "" {
			t.Errorf("%s: a misuse wrote to stdout: %q", c.name, stdout)
		}
	}
}

// The usage text is the only place a reader learns what the command can do, so
// a subcommand missing from it is a subcommand nobody finds.
func TestUsageNamesEverySubcommand(t *testing.T) {
	var buf bytes.Buffer
	usage(&buf)
	for _, c := range commands() {
		if !strings.Contains(buf.String(), c.name) {
			t.Errorf("usage does not name %q", c.name)
		}
		if !strings.Contains(buf.String(), c.summary) {
			t.Errorf("usage does not summarise %q", c.name)
		}
	}
}

// The set is fixed at two here on purpose. This is the scaffolding commit and
// the plan adds subcommands one issue at a time; a third arriving without its
// own issue is what this notices.
func TestTheSubcommandSetIsTheTwoThisCommandDeclares(t *testing.T) {
	var got []string
	for _, c := range commands() {
		got = append(got, c.name)
		if c.summary == "" {
			t.Errorf("%q has no summary", c.name)
		}
		if c.run == nil {
			t.Errorf("%q has nothing to run", c.name)
		}
	}
	want := []string{"version", "help"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("subcommands are %v, want %v", got, want)
	}
}
