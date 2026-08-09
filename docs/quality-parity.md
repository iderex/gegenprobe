# Quality parity against the target gate

The gate that runs on https://github.com/iderex/jellyfin-plugin-sso is the target
for this board's own gate. That board is public, its workflows are readable, and
its checks can be listed rather than described, which is what makes it a target
instead of an aspiration.

This file maps every check there onto this board and says, for every row nothing
here covers, why the absence is a deviation rather than a gap.

Target list taken: 2026-08-09
Required status checks: none

The two fields above are read by the `quality parity` leg of `go run ./gate`
rather than by a person. The first says when the target list was last derived.
The second says which checks can stop a merge here, and the leg refuses a `yes`
in the table's merge column for any check the field does not name.

## Where the target list comes from

The list is derived rather than remembered, and both heads have to be read. Some
checks appear only on a pull request head and others only on the default branch
head, so a table taken from either alone is short.

```
gh api repos/iderex/jellyfin-plugin-sso/commits/main --jq .sha
729d2ecab3f3377727c61e9adfdbef58fc737266
gh api "repos/iderex/jellyfin-plugin-sso/commits/729d2ecab3f3377727c61e9adfdbef58fc737266/check-runs?per_page=100" --jq '.check_runs[].name' | sort -u
ABI floor build
Analyze (actions)
Analyze (csharp)
Analyze (javascript-typescript)
Audit workflows (zizmor)
build
Enforce greppable invariants
Package (JPRM) / Build package
Package (JPRM) / Generate SBOM
prettier
Reject Trojan Source Unicode
Report any workflow that concluded non-success on the default branch
Scorecard analysis
submit-nuget
wiki-lint
```

```
gh api "repos/iderex/jellyfin-plugin-sso/commits/4e19cb5e557fa0d5b9792ae678ac84773d8fe97b/check-runs?per_page=100" --jq '.check_runs[].name' | sort -u
ABI floor build
Analyze (actions)
Analyze (csharp)
Analyze (javascript-typescript)
Audit workflows (zizmor)
build
CodeQL
DCO sign-off
dependency-review
Deterministic PR-hygiene checks
Enforce greppable invariants
Package (JPRM) / Build package
Package (JPRM) / Generate SBOM
prettier
Reject Trojan Source Unicode
submit-nuget
zizmor
```

Twenty distinct names across the two. `CodeQL`, `DCO sign-off`,
`dependency-review`, `Deterministic PR-hygiene checks` and `zizmor` are on the
pull request head only. `Report any workflow that concluded non-success on the
default branch`, `Scorecard analysis` and `wiki-lint` are on the default branch
head only.

Scheduled work publishes nothing on either head until it has run against that
commit, so the last six rows come from the workflow list instead, filtered to
the ones carrying a schedule whose name reaches neither head:

```
gh api repos/iderex/jellyfin-plugin-sso/actions/workflows --paginate --jq '.workflows[].name' | sort
```

Both boards move. Re-run the three commands rather than trusting the output
pasted above, which is what it said on the date in the field at the top of this
file and makes no claim about any other day.

## Two names that are not workflow jobs

`CodeQL` and `zizmor` on the target's pull request head are published by code
scanning rather than by Actions, and no job in any workflow file on either board
is called either name. The same shape is live here, which is why a leg reading
the workflow files alone would not find them:

```
gh api "repos/iderex/gegenprobe/commits/f22043b33b98cdeebef500cafe8d4cdeba641ff6/check-runs?per_page=100" --jq '.check_runs[] | "\(.name)\t\(.app.slug)"' | sort
Analyze (actions)	github-actions
Analyze (go)	github-actions
Audit workflows (zizmor)	github-actions
CodeQL	github-advanced-security
Commit hygiene	github-actions
Cross build	github-actions
DCO sign-off	github-actions
dependency-review	github-actions
Gate	github-actions
Race	github-actions
Reject Trojan Source Unicode	github-actions
Reject Trojan Source Unicode	github-actions
zizmor	github-advanced-security
```

Thirteen distinct names across thirteen runs, one of which is published twice.
`Reject Trojan Source Unicode` runs once per event because
`.github/workflows/unicode-guard.yml` declares both `push` and `pull_request` on
every branch, so a branch pushed here and then opened as a pull request runs that
job twice on one commit. Six runs under five names was the earlier count and it
was taken before this board's own gate existed.

## How to read a row

The covering column holds `-` where nothing here covers the check, or a
semicolon separated list of entries. An entry is either `leg <name>`, naming a
leg of `go run ./gate`, or `workflow <path>`, naming a workflow file in this
tree. The leg refuses an entry that resolves to neither.

A row carrying both a covering entry and a note is a partial cover, and the note
says which part is missing. A row carrying neither a covering entry nor a reason
is refused, because a blank cell and a considered absence read the same on the
page and are opposite statements.

The merge column says whether that check can stop a merge here. Every row says
`no` today, and that is not a shorthand for nothing having been decided: the
ruleset on the default branch carries no required status checks at all, which is
the `Required status checks` field at the top of this file. Which checks become
preconditions is #82, and it cannot state the ruleset's final shape until an
entry of #1 is answered.

## The table

| Check on the target board | What it covers there | Covered here by | Blocks a merge here | Issue | Note, or the reason for the deviation |
| --- | --- | --- | --- | --- | --- |
| ABI floor build | that the built assembly still loads against the oldest host application version the plugin supports | - | no | - | nothing here loads into a host application, so there is no host ABI to hold. The compatibility question this board has instead is which platforms the binary builds for, and the Cross build job in `.github/workflows/ci.yml` answers it |
| Analyze (actions) | code scanning over the workflow definitions | workflow .github/workflows/codeql.yml | no | #59 | - |
| Analyze (csharp) | code scanning over the sources of the language the board is written in | workflow .github/workflows/codeql.yml | no | #59 | the language here is Go, so the job publishes `Analyze (go)`. The row is the same check under a different language rather than a deviation |
| Analyze (javascript-typescript) | code scanning over the JavaScript and TypeScript sources | - | no | - | there is no JavaScript or TypeScript in this tree, and the only two languages the analysis is given here are Go and Actions |
| Audit workflows (zizmor) | the workflow audit reported as a job, so it is red on the pull request page | workflow .github/workflows/zizmor.yml | no | - | - |
| build | that the tree compiles and its unit tests pass | leg tests; leg vet; leg coverage; workflow .github/workflows/ci.yml | no | #68 | - |
| CodeQL | the code scanning result of the analysis jobs, published by the security product | workflow .github/workflows/codeql.yml | no | #59 | - |
| DCO sign-off | every commit carrying a Developer Certificate of Origin sign-off | workflow .github/workflows/dco.yml | no | - | - |
| dependency-review | a dependency added in a pull request, against known vulnerabilities and a licence policy | workflow .github/workflows/dependency-review.yml; leg dependencies | no | #64 | - |
| Deterministic PR-hygiene checks | the shape of the pull request itself, its title, its body and its commit subjects | leg commit hygiene | no | #65 | the commit subjects and their issue references are covered. The title and the body of the pull request are read by nothing here |
| Enforce greppable invariants | patterns this repository refuses, one rule per decision behind it | - | no | #60; #67 | nothing in this tree performs it today. The two legs nearest to it, `package boundaries` and `documented paths`, judge the import graph and the documents rather than the source, and the rules #60 names are mostly about packages that are not written yet |
| Package (JPRM) / Build package | building the distributable plugin package | - | no | #63 | the artefact here is a binary rather than a plugin package, so there is no packaging step for a check to run. Checksums, an SBOM and provenance over the binary cover the same ground |
| Package (JPRM) / Generate SBOM | an SBOM for the built package | - | no | #63 | the same reason, one step further on. There is no package to describe, and the SBOM here belongs with the release artefacts rather than with a build |
| prettier | formatting of the web assets | - | no | #69 | this board has no web assets. Formatting is `leg format` for the Go source and `leg documentation form` for the Markdown |
| Reject Trojan Source Unicode | bidirectional and invisible Unicode in tracked text | workflow .github/workflows/unicode-guard.yml | no | - | - |
| Report any workflow that concluded non-success on the default branch | a sweep raising an alert where a workflow failed on the default branch | - | no | #66 | there is nothing here to watch yet. One scheduled workflow exists in this tree and the four runs #66 names as its subject do not |
| Scorecard analysis | the supply chain score, published weekly and on the default branch | workflow .github/workflows/scorecard.yml; leg supply chain | no | #85 | the run publishes a score and `docs/supply-chain.md` answers every check it reports. What the leg reads is the output recorded in that file rather than a live score, and the file says so in its own last section |
| submit-nuget | publishing the built package to a package registry | - | no | #63 | nothing here is published to a registry. The release artefact is a binary, and what stands behind it is a checksum and provenance rather than a registry's own acceptance |
| wiki-lint | the project wiki, which is outside the tree | - | no | #69 | this board has no wiki. Its documentation lives in the tree, where three legs read it: `documentation form`, `documentation links` and `documented paths`. Links that leave the tree are read on a schedule by `.github/workflows/external-links.yml` |
| zizmor | the code scanning result of the workflow audit | workflow .github/workflows/zizmor.yml | no | - | - |
| Fuzz (SharpFuzz) | a weekly fuzz run over the parsers | - | no | #61 | there is no parser here to fuzz yet. The readers this would be pointed at are the milestone before this one |
| Stryker mutation testing | a weekly mutation run over the whole product | - | no | #62 | the two packages the run is scoped to here are not written. The scope is also narrower than the target's on purpose, because the mutation tooling for this language is less mature and a run over the whole tree would produce noise nobody triages |
| E2E Login Harness | a daily end to end run of the product's main path against a real deployment | - | no | #41 | the equivalent here is a run of a real code inside a real container engine, which is the integration harness and is deliberately outside the gate |
| Manifest freshness | a daily assertion that the published manifest lists the newest release per generation | - | no | #74 | nothing here publishes a manifest, and what a release consists of is not decided yet |
| Performance baseline (login latency) | a weekly latency measurement against a recorded baseline | - | no | - | there is no interactive path here to time. The runtime of a run is dominated by the codes the harness starts, which are other people's programs, and holding them to a baseline in this repository would measure the machine rather than the change |
| Nightly betas | a nightly dispatch of beta builds | - | no | #74 | nothing here is released nightly. What a version means and when the first release happens are the release milestone's, not this one's |

## Issues in this milestone that close no row

Three do not map onto a check on the target board, and listing them here rather
than forcing a row keeps the table one row per target check.

This file is #58. The merge column is #82, which decides which checks become
preconditions and brings the ruleset to match. #83 runs the gate suite on a
machine where the capabilities the tests are forbidden to reach are actually
absent, which is a statement about where a suite runs rather than about which
checks exist. It landed as `.github/workflows/tier-without-capabilities.yml`,
kept honest by the leg `capability absence job`, and it is the same tier as the
`build` row above rather than a check beside it.

Where that job meets a runner it cannot take a capability away from, it prints
which one, and the line belongs here. There is none today, and the section
below is where one would go.

## What the capability absence job could not establish

Nothing so far. Each line here names a capability
`.github/workflows/tier-without-capabilities.yml` could not remove on the runner
it was given, so that a green tick on that job is read as covering the rest and
not this. An empty section is the claim that the run established all four, and
it is checked by reading that job's own output rather than by anything in this
tree.

## What this table does not do

It reads in one direction. A row naming a covering leg or workflow that does not
exist is refused, and a workflow file in this tree that no row mentions is
refused, so the table cannot half rot against the tree. What it does not refuse
is a check published on this board that no row mentions, because the published
name of a job carrying a matrix is not derivable from the workflow file without
reading the matrix, and the two code scanning names above are published by no
job at all. That direction is #82's fourth condition and it is not carried here.

The merge column is checked against the `Required status checks` field at the
top of this file and not against the ruleset itself. Nothing in this tree reads
the ruleset, and a leg that reached for it would be a gate leg making a network
call. So the field is a declaration a person keeps true, and the agreement
between it and the live ruleset is what #82 still owes. Until then a `yes` in
that column is refused, which fails in the safe direction: the table cannot claim
a check blocks a merge while the declaration says none does.
