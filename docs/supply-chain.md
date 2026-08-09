# The supply chain score, one outcome per check

`.github/workflows/scorecard.yml` runs the OpenSSF Scorecard against this
repository and publishes the result, so a number and a code scanning upload exist
whether or not anybody acts on them. A published score with nothing behind it is
read as a verdict somebody reached, and the workflow's own header calls the score
a checklist rather than a guarantee. This file is the checklist.

Every check the recorded output names has one outcome here: `fixed`, `accepted`
with a reason and the condition that would retire the acceptance, or
`not applicable` with a reason. A check with no row is refused by the
`supply chain` leg of `go run ./gate`, and so is a row naming a check the
recorded output does not hold.

It sits beside `docs/quality-parity.md` on purpose. That file says which checks
this board runs against the gate it is measured by; this one says what the
supply chain posture is. A reader looking for either meets both.

Output taken: 2026-08-09
Scorecard version: v5.5.0
Scored commit: 772e86784f782d945c0702b1fd20699f8473c024

The three fields above are read by the leg rather than by a person. The first
says when the output below was recorded, the second which version of the tool
produced it, and the third which commit was scored.

## How the output is recorded

The score is published to the OpenSSF API by the workflow, and that published
score is what this register is written against, because it is the one a reader
finds without asking this repository for anything.

```
curl -sS https://api.securityscorecards.dev/projects/github.com/iderex/gegenprobe > scorecard.json
python -c "import json; d = json.load(open('scorecard.json')); print(d['scorecard']['version'], d['repo']['commit']); [print(c['name'], c['score']) for c in d['checks']]"
```

The first line that prints is the version and the commit in the two fields
above. The rest is one line per check, and it is what the next section holds.

## The recorded output

The first fenced block under this heading is what the leg reads. One line per
check, its name and its score, exactly as the commands above printed them.

```
Packaging -1
Code-Review 0
Dangerous-Workflow 10
Token-Permissions 10
Maintained 0
Binary-Artifacts 10
Pinned-Dependencies 8
License 0
CII-Best-Practices 0
Vulnerabilities 10
SAST 10
Fuzzing 0
Dependency-Update-Tool 0
Signed-Releases -1
Branch-Protection 3
Contributors 0
Security-Policy 0
CI-Tests 10
```

Eighteen checks. A score of `-1` is the tool saying the check did not apply
rather than saying the repository failed it, and the two rows carrying one below
are `not applicable` for that reason.

## How to read a row

The outcome column holds one of three words and nothing else.

`fixed` means the check passes and the thing it asks for is in the tree. The
reason column says what it is, so the row is still readable when the score
moves.

`accepted` means the check does not pass and this project is not going to make
it pass yet. The row names the condition that would retire the acceptance, which
is what keeps this a list of debts rather than a list of dispensations. The leg
refuses an acceptance whose retirement cell is empty.

`not applicable` means the check is about something this project does not have
and will not acquire by fixing anything. The retirement cell is `-`, because
there is no debt to retire, and the leg refuses a retirement condition on a row
that is not an acceptance.

The score column is the score in the block above. The leg refuses a row whose
score disagrees with it, so re-recording the output without revisiting the
triage is refused rather than passing quietly.

The issue column names the issues that hold the work, or `-`. Where the argument
is already made in an issue, the reason column points at it instead of making it
again.

## The register

| Check | Score | Outcome | What retires the acceptance | Issue | Reason |
| --- | --- | --- | --- | --- | --- |
| Packaging | -1 | not applicable | - | #63 | nothing here is published to a package registry, so there is no packaging workflow for the check to find. The release artefact is a binary, and what stands behind it is a checksum, an SBOM and provenance |
| Code-Review | 0 | accepted | entry 8 of #1 is answered and #82 brings the ruleset to match the answer | #1; #82 | the ruleset on the default branch requires no approving review, so nothing that has landed carries one the check can count. Whether it should is a maintainer decision and not the plan's |
| Dangerous-Workflow | 10 | fixed | - | - | the workflow audit in `.github/workflows/zizmor.yml` reads every workflow in this tree and publishes its result as a job, so a dangerous pattern is red on the pull request page as well as in the score |
| Token-Permissions | 10 | fixed | - | - | every workflow declares read-only permissions at the top level and grants a write scope only on the job that needs it |
| Maintained | 0 | accepted | the repository is older than the ninety days the check measures, which no change to the tree brings forward | - | the check reports that the repository was created inside the last ninety days. It is a statement about the repository's age and about nothing in it |
| Binary-Artifacts | 10 | fixed | - | - | no binary is committed to this tree, and the only executable it produces is built from its own source |
| Pinned-Dependencies | 8 | accepted | the analyser is fetched by commit sha, or it stops being fetched at all because static analysis moves inside this module | #19 | the one fetch not pinned by hash is the `go install` of the static analyser in `.github/workflows/ci.yml`. The version there is the one the gate's own skip message names, and pinning by sha instead would leave two spellings of that version to keep in step |
| License | 0 | accepted | entry 1 of #1 is answered, and #25 lands the licence file and the header check against it | #1; #25 | there is no licence file. Which licence this repository is offered under is a maintainer decision, and nothing here can choose one on its own |
| CII-Best-Practices | 0 | accepted | the project registers for the badge and answers its questionnaire | - | the badge is held on another site and earned by answering a questionnaire there. Nothing in this tree changes the score, and the evidence the questionnaire asks about is the gate, the decision records and this file |
| Vulnerabilities | 10 | fixed | - | - | the check finds no known vulnerability in what this repository depends on, which is currently nothing outside the standard library |
| SAST | 10 | fixed | - | #59 | code scanning runs over the Go sources and over the workflow definitions, in `.github/workflows/codeql.yml` |
| Fuzzing | 0 | accepted | #61 lands a fuzz target per reader, seeded from that reader's fixtures | #61 | the surface worth fuzzing here is the readers, which parse files produced by programs this project does not control, and no reader is written yet |
| Dependency-Update-Tool | 0 | accepted | `go.mod` acquires a direct requirement | #64 | the answer is below, in the section this check has to itself |
| Signed-Releases | -1 | not applicable | - | #63; #75 | there is no release. The check has nothing to look at, and what a release consists of here is #63 |
| Branch-Protection | 3 | accepted | entry 8 of #1 is answered and #82 brings the ruleset to match the answer | #1; #82 | the four warnings are approvals, stale review dismissal, last push approval and required status checks, and all four are the same question: which of this board's checks become preconditions of a merge, and whether a second reader is one |
| Contributors | 0 | accepted | the project accepts a contribution from outside, which cannot happen before entry 1 of #1 is answered | #1 | the check counts contributors affiliated with an organisation over the last year. Nothing has been contributed from outside, because there is no licence for a contributor to contribute under |
| Security-Policy | 0 | accepted | #26 lands the security policy, saying what entry 6 of #1 decides it says | #1; #26 | private vulnerability reporting is switched on at the repository, and no file in the tree tells a finder that. The check reads the file rather than the setting, and so does somebody with a finding |
| CI-Tests | 10 | fixed | - | - | the check found a test run on every merged pull request it examined. The jobs it found are the gate, the race detector run and the cross build, declared in `.github/workflows/ci.yml` |

## The dependency update question

Scorecard rewards a configuration file in the tree that opens version bump pull
requests. There is none, and the answer is that there will not be one before the
first release.

Two facts nearby are easy to mistake for one. Dependabot security updates are
switched on at the repository, which reacts to an advisory and opens nothing
otherwise. The `dependency-review` job reads the diff of a pull request that
changes a dependency. Neither of them is the tool this check is asking for.

The reason for the answer is that the surface such a tool would keep current is
small enough to read. The module has no direct requirement at all:

```
go list -m all
github.com/iderex/gegenprobe
```

What is versioned instead is the actions the workflows use, every one of them
pinned to a commit sha with the version in a comment beside it, and the leg named
`action pinning` refuses one that is not. A bot opening a pull request per action
per release would be most of the traffic on this board and none of the work.

What the project does instead of running one: `docs/dependencies.md` argues each
direct dependency before it is taken and the `dependencies` leg refuses a module
file the argument does not cover, `dependency-review` reads what a change adds,
and the actions are pinned by sha where a moved tag cannot reach them. What none
of that gives is the thing the check is really about, which is finding out that
something has moved without somebody going to look.

The condition that retires this acceptance is `go.mod` acquiring a direct
requirement. At that point the surface stops being one file of pinned actions and
the argument above stops holding.

## What this leg does not do

It does not run Scorecard and it does not read a live score. The block under
`## The recorded output` is a recording somebody pasted, and the leg compares the
register against that recording. Both are files in this tree, so the comparison
is between two things a person edits, and what it refuses is half an edit: a
recording brought up to date without the triage being revisited, or a row written
for a check the recording does not name.

Nothing refuses a stale recording. The `Output taken` field says when it was
made and no leg reads the date against anything, because a gate leg that fetched
the live score would be a gate leg making a network call, which this tree does
not have and the gate tier's own capability check refuses in the tests beside it.
Re-recording is an edit somebody makes, and the same is true of the target list
in `docs/quality-parity.md` for the same reason.

It reads the score and not the finding behind it. Scorecard's SARIF carries a
message per check saying which file and which line it objected to, and this
register carries the score and a reason written by a person. Where the two would
disagree, the SARIF is the tool's word and this file is the project's answer to
it.
