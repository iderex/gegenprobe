# 0009. Three test tiers, and the gate tier needs no display, no elevation and no network

Number: 0009
Title: Three test tiers, and the gate tier needs no display, no elevation and no network
Status: accepted
Date: 2026-08-07

## What was decided

Three tiers, named rather than numbered, because a number invites somebody to
treat the second one as the first one with more of it.

### The gate

Unit and contract tests over fixtures committed to this repository. It is what
the single check command runs, it is what stands behind a merge, and it is the
only tier any other route is allowed to call the tests.

It may depend on: the Go toolchain, this repository's own source, files under a
`testdata` directory reached by a relative path, and a temporary directory the
test framework handed it.

It may not depend on: a container engine, a display, elevated rights, a network
of any kind including loopback, a code from outside this tree, a file outside the
repository and outside its own temporary directory, an environment variable it
did not set itself, or the machine's clock beyond monotonic elapsed time.

It carries no build tag. A test file with no tag is a gate test, so the default
is the strict tier and a contributor has to act deliberately to leave it.

### The integration harness

Tests that start a real code in a real container and check the runner contract,
the recipes and the readers against output the codes actually produced. It is
never called a unit test and never called the gate, in a file name, a package
name, a job name or a sentence.

It may depend on: a container engine the operator installed, code sources the
operator supplied, images built from this repository's recipes, and a writable
working directory.

It may not depend on: elevated rights, a display, or a network reaching anything
other than the registries and source locations the operator named. Rootless is
the requirement 0003 already imposes on the recipes, arriving here as a
requirement on the tests that exercise them.

It carries the build tag `integration` and is opt in. Where its preconditions are
absent it does not pass quietly. It prints, per precondition, that it was not
asked for and what asking would have cost, and the run's summary names every test
that did not execute. A run that covered less than the whole tier cannot be read
as one that covered it and found nothing.

### Physics regression

A small fixed set of cases run end to end against stored bundles, taking minutes
to hours. It answers the question no smaller test can: whether the numbers this
project publishes have moved.

It may depend on everything the integration harness may depend on, plus stored
bundles from previous runs held in the repository or fetched from a location the
operator named, and plus as much time as the codes take.

It may not depend on anything the integration harness may not, and it may not run
on every change. It runs on a schedule and on demand. It carries the build tag
`regression`.

### Where a new test goes when it is unclear

Default to the gate. If the thing it needs can be replaced by committed bytes,
replace it and stay. If it cannot run in the gate tier without one of the
forbidden capabilities below, it belongs to the integration harness. If it needs
several codes, a whole case and a stored bundle to say anything, it belongs to
physics regression.

The tie break is one sentence: if two people are arguing about whether a test
belongs to the gate, it belongs to the gate, and whatever it wanted has to become
a fixture. Where that turns out to be impossible, the impossibility is the answer
and it is written into the test's own comment, naming what could not be
fixtured.

### The forbidden capabilities, in terms a machine can check

The list is written so it can be refused statically over the gate tier's source,
which is the check planned in the scaffolding milestone as issue #21, and then
confirmed by running the gate suite where the capabilities are actually absent,
which is issue #83. Both are needed: the static half catches the case before it
runs anywhere, the executed half catches what the static half cannot see through.

A gate tier file is any Go file under this module whose build constraints do not
include `integration` or `regression`. Over that set, all of the following are
refused.

- **Any import of `os/exec`.** This is one rule covering elevation, installers,
  service registration, certificate trust stores, container engines and every
  external binary, rather than a list of program names that would always be one
  name short. A gate test that wants to run a program is asking for the
  integration harness.
- **Any import of `net`, `net/http`, `net/rpc`, `net/smtp`, `net/http/httptest`,
  `crypto/tls`, or any package outside the standard library that is not in this
  module's own import graph for non-test code.** Loopback is included: a test
  server on `127.0.0.1` binds a socket, and a machine that forbids that is one
  this suite has to survive. `net/url` is not on the list, because it parses text
  and opens nothing.
- **Any read of `DISPLAY`, `WAYLAND_DISPLAY`, `XAUTHORITY` or `BROWSER`**, and
  any import of a package that opens a windowing connection. A gate test never
  needs a screen, and a test that reads these to decide whether to skip is
  deciding to be conditional, which is the shape this tiering exists to remove.
- **Any call to `os.UserHomeDir`, `os.UserCacheDir`, `os.UserConfigDir`,
  `os.TempDir`, `os.MkdirTemp`, `os.Getwd`, or `os.Chdir`.** The only directory a
  gate test writes in is the one `t.TempDir` returned, which the framework
  removes afterwards, and the only directory it reads from is a relative
  `testdata` path.
- **Any absolute path literal**, matched as a string constant beginning with `/`
  or with a drive letter followed by `:\` or `:/`. A path of the test's own
  choosing is the failure this covers, and it covers the platform specific
  spellings because this suite runs on more than one.
- **Any call to `os.Setenv`, `os.Unsetenv` or `os.Clearenv`.** A gate test that
  needs an environment variable uses `t.Setenv`, which restores it, so no test
  can change the environment another test reads.
- **Any use of `syscall` or `golang.org/x/sys` from a test file.** Privilege
  requests, capability changes and platform specific mount work all arrive
  through these, and none of them belongs in the tier that has to run on a
  managed desktop.

The list is a denylist and denylists are incomplete by construction. That is why
issue #83 exists: the executed half runs the gate suite with no display variable
set, with the network unavailable, with the home directory unwritable and with
the working tree read only, and requires it green. A capability nobody thought to
deny still fails there.

A change that makes a gate test need any of these is a defect in the change, not
a reason to widen the list. Widening it is a decision that supersedes this
record.

## Why

A gate that cannot run on a laptop, on a managed desktop, or in a plain CI
container is a gate that gets skipped, and a skipped gate is worse than none
because everyone believes it ran.

The skipped-reads-as-passed failure is the one this tiering exists to prevent,
and it is not hypothetical. A suite that quietly returns success when its
container engine is missing will be green on the machine where it matters least,
and the person reading that green has no way to tell it from the other kind.
Making the integration tier announce its own absence, by name and per
precondition, is the whole reason the tiers are named rather than counted.

The gate tier can be complete about the parts this project actually writes. The
readers, the canonicalisation, the identification, the comparison, the report
rendering and the significance rules are all functions over bytes, and bytes can
be committed. What genuinely needs a code running is the runner contract and the
recipes, and that is a much smaller surface than it looks like from the outside.

The elevation rule is a birth requirement rather than something to fix later.
Adding it after the fact means auditing every test that already exists and
arguing with each one, and the codes this project drives are exactly the sort of
software that tempts a developer into a privileged container. It is also a rule
about the person at the machine: a test that raises a consent prompt interrupts
whoever is sitting there, and an interruption is not something a test suite gets
to cause.

Untagged means gate is a small choice with a large effect. The strict tier is
what a contributor gets by not thinking about it, and leaving it requires typing
a line that says which tier they meant.

## What was rejected

A single suite with skips. It is what most projects do, and it is how a suite
ends up green on a machine that ran a third of it.

Requiring a container engine for the whole suite. Simplest to write, and it makes
the gate unrunnable in the environments this project most wants contributors
from, which include the managed and air-gapped machines where this work happens.

Generating fixtures at test time by running the codes. Removes the need to commit
them, and makes every test depend on the thing the tests exist to check.

Two tiers, folding physics regression into the integration harness. Fewer
conventions, and it puts an hours-long run behind a tag somebody will eventually
put in a pull request check, at which point either the check is disabled or the
tier is.

An allowlist of permitted imports instead of a denylist. Stricter and better in
principle, and it turns every ordinary standard library addition into a change to
the checking rule, which is how a strict rule gets switched off.

## What this costs

Fixtures have to be obtained, trimmed, committed and given a recorded provenance,
and trimming output from a code that writes hundreds of megabytes is work with
its own risk of cutting away the part that mattered.

Whether recorded output from every code may be committed at all is not settled
here. It is entry 3 of the maintainer decisions issue. The fallback if the answer
is no is synthetic fixtures in the same fixed format layout carrying invented
numbers: they test the readers just as well and they prove nothing about physics.
That is a real loss and it is why it is a fallback rather than a plan.

Three suites mean three sets of conventions and three ways for a contributor to
be confused about where their test belongs. The tie break sentence above is the
mitigation and it is a sentence, not a mechanism.

The denylist will refuse something legitimate. When it does, the answer is a
narrower rule or a different tier, argued in an issue, and not a suppression
comment on the line. A suppression that lands without an issue is how the list
becomes decorative.

Nothing in this record is enforced today. Issue #21 owes the static half and
issue #83 owes the executed half, and until both land a gate test that opens a
socket reaches the default branch exactly as one that does not.
