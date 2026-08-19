# Security policy

## What this repository is, so that a report can be aimed at it

The README describes a harness that runs several atomic structure codes against
the same case and reports where they agree. That is what this project is for,
and it is not yet what the tree holds. A policy written against the description
would be describing software nobody can run.

What is here today is a Go module with no direct dependencies at all. The only
third-party code in a build of it is the Go standard library. The command it
builds reads its arguments, answers `version` or `help`, and exits;
`internal/cli/cli.go` is the whole of that behaviour, and its own usage text
says that nothing here runs a case yet. The rest of the tree is the machinery
around that command: a bundle model, a reader contract with no reader registered
behind it, the decision records the design is argued in, and a quality gate that
reads this repository and refuses it where it disagrees with those records.

There are no tags and no releases. Nothing built from this repository is in
anybody's hands, so there is no shipped artefact to carry a vulnerability, and a
report is against `main` at the commit you read.

## Reporting

Private vulnerability reporting is switched on for this repository, so the
advisory form is the channel and it answers today:

<https://github.com/iderex/gegenprobe/security/advisories/new>

I checked that rather than assuming it:

```
$ gh api repos/iderex/gegenprobe/private-vulnerability-reporting
{"enabled":true}
```

What makes a report usable here is the commit you read, the file and the line,
and the input that reaches the code you are describing. A recorded file is worth
more than a description of one.

I promise no acknowledgement deadline, and I am not going to invent one. A
reporter who is told to expect an answer inside a stated time and does not get
it learns nothing about their report and cannot tell a slow week from a message
that never arrived. Silence with no promise attached is at least honest about
what it is. This project exists to take unenforced claims out of atomic data, so
it should not open with one of its own.

## What I would treat as a vulnerability here

**Bytes this project did not produce.** `internal/model/read.go` reads a bundle
manifest that some other run wrote, probes its format version and refuses a
format this build does not know. A crash, an unbounded allocation driven by the
document, or a manifest accepted under a version it was not written for, is a
real finding, because a bundle is meant to travel between machines and be reread
long after the build that made it is gone.

**The reader contract.** `internal/reader/contract.go` judges whether a reader
of another code's output keeps seven requirements. One of them,
`a-file-from-another-code-is-refused-rather-than-parsed`, is a security property
wearing a physics name: these formats are near enough to each other that the
wrong reader turns a foreign file into plausible numbers instead of an error. No
reader is registered in this build, so what exists to attack today is the judge,
and a reader that breaks a requirement and passes `Check` anyway is worth an
advisory, because every reader that ever lands is judged by it.

**The one thing in this tree that opens a socket.** `tools/externallinks` walks
the Markdown, extracts the URLs written in it, and requests each one with
`net/http`. It runs weekly from `.github/workflows/external-links.yml`, where
its job holds `issues: write` and a token, and files a tracking issue whose body
carries the report text. Two shapes interest me: a way to make that job use its
token for anything but the single tracking issue, and a way to make the reader
reach a host or a resource that is not a link written in this tree, whether
through the extraction or through a redirect it follows.

**The workflows themselves.** Every action is pinned to a commit sha,
permissions are denied at the top and granted per job, checkouts run with
`persist-credentials: false`, and nothing here uses `pull_request_target`. If
you find a route that gets a token, a secret or contributor-controlled text into
a shell in `.github/workflows`, I would much rather hear it privately than read
it in a log. Two things there are fetched over the network by version rather
than by hash: the static analyser installed in `ci.yml`, and the workflow
auditor installed in `zizmor.yml`. The first is written down as an accepted gap
in `docs/supply-chain.md` under Pinned-Dependencies. Restating either is not a
report; showing a route through one that the record does not anticipate is.

**The tools a contributor runs on a checkout.** `go run ./gate`,
`tools/commithygiene` and `tools/decisionindex` read the tree they are pointed
at, including test fixtures and source files. Checking out a branch and running
the gate should not execute anything the author of that branch chose. If it can,
that is a vulnerability in this repository even though the gate never ships.

## What is not a vulnerability here

**A number this project gets wrong, or two codes disagreeing.** That is the
output, not a fault in it. `.github/ISSUE_TEMPLATE/suspected-disagreement.yml`
is where that goes, in the open, because the reasoning behind it is the point.

**A defect in one of the codes being compared.** This repository ships no source
and no images of AUTOSTRUCTURE, FAC, Cowan's chain or RMATRX1, and decision
record 0003 says it never will. I have nothing to patch and no standing to patch
it, so those go to the people who wrote them. A build recipe in this tree is a
different matter: a recipe that weakens the isolation a code runs under, or that
asks for elevation, is mine, and 0003 already calls the second one a defect.

**The absence of authentication, sandboxing or hardening around the program.**
It is a command line tool an operator runs on their own machine, under their own
account, on files they chose. There are no users, no listening socket and no
privileged component, so there is no boundary inside it for anybody to cross,
and anything that needs the operator's account already is not a finding.

**A vulnerability report against a dependency.** There are none.
`go list -m all` prints this module and nothing else. An advisory about the Go
toolchain belongs to the Go project, and reaches this repository only if the way
this code uses the standard library is what makes it reachable.

**A Scorecard row that `docs/supply-chain.md` already records as accepted.** No
required approving review, no fuzz targets yet, no dependency update tool: each
of those carries the condition that retires it, written down with its reason. A
report that restates one is telling me something I wrote.

**Missing signing or provenance on a release.** There is no release. What a
release consists of is still being decided, and that argument is on the tracker.

**A published bundle that is identifiable, but not through a field record 0012
removes.** Redaction takes out host names, user names and absolute paths. It
does not make a bundle anonymous, and the record says so rather than claiming
otherwise: a case name or a configuration label somebody chose can name their
institution, and the pre-publication report shows it instead of hiding it. A
bundle that keeps a host name, a user name or an absolute path after redaction
would be a real finding, once that command exists.

**A hole in a component that is not written yet.** The container runner, the
publish and redaction command of record 0012, and the Python fit component of
record 0014 are designs on the tracker rather than code in the tree. I want
those arguments, and I want them in the open where they can be answered, not in
a private advisory about software nobody can run.
