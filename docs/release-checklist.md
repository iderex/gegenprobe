# The release checklist

The steps worked through before a release is tagged, in order. It is written
before the first release rather than reconstructed from one, so that the first
release is judged against something it did not get to choose.

Every step says what makes it done. Where a step cannot be answered yet because
the thing it asks about is unbuilt, it names the issue that will make it
answerable and stays in the list, because a checklist that drops such a step gets
shorter every release and nobody notices which control left.

Three steps below have no artefact behind them today. They are marked in place
rather than in a footnote.

## Before the tag

### The gate is green on the commit being released

```
go run ./gate
```

The last line says how many legs ran and how many examined nothing. A leg that
was skipped covered nothing, so a run with a skip is not a run that passed the
whole set, and the release records which leg and why. A green run on a different
commit is not evidence about this one.

### The regression suite has run

Not answerable yet. The physics regression over the example cases is #57 and
there is nothing to run, so this step is answered with that sentence and not with
a tick. The same applies to the integration harness in #41: the gate tier the
step above measures deliberately excludes both, which `docs/coverage-floor.md`
states beside the number it prints.

### The documentation describes the tool being released

`README.md` says what the tool does, not what it will do. The operator
documentation from #71 gets somebody from an empty machine to a first map using
the version being released. `docs/quality-parity.md` carries a target list taken
recently enough to be worth quoting, since that board moves. Every link and every
path the tree's Markdown names resolves, which the `documentation links` and
`documented paths` legs decide rather than a reader.

### The register of unsupported participants is current

A map covering fewer codes than a reader assumes is the worst artefact this
project could produce, so the register of participants that cannot be fully
supported is rechecked and its dates moved before the release rather than after.
That register is #42 and does not exist yet.

### The changelog entry is written

`CHANGELOG.md` has a section for this version, dated, with every line naming its
issue, and with the section about bundles somebody already holds filled in rather
than omitted. Writing it after the tag means writing it from the diff, which is
how the one sentence a bundle holder needs goes missing.

### The three version numbers agree with what changed

`docs/versioning.md` says what each of the tool version, the case schema version
and the bundle schema version means. The release states all three, including the
two that did not move, because an unstated number reads as an unchanged one and
only one of those is a claim.

## The tag and the artefacts

### The tag is signed and names the version

The tag is `vMAJOR.MINOR.PATCH` and is signed the way commits here are signed.
Nothing in this repository refuses an unsigned tag, so this step is held by
whoever performs it.

### The build stamps the tag

```
go build -ldflags "-X github.com/iderex/gegenprobe/internal/version.stamped=v0.1.0"
```

Then the binary is asked what it is, and the answer is the tag rather than a
commit or a sentence about missing metadata:

```
gegenprobe version
```

### The build is reproducible from the tag

Two builds of the tagged tree produce identical bytes and neither carries the
directory it was built in. The gate has a leg for that, `reproducible build`, and
this step is it run against the tag rather than against a branch.

### The release artefacts are attached

Not answerable yet. Checksums, an SBOM and provenance are #63 and none of the
three exists, so a release made before it lands says plainly which artefacts it
does not carry instead of attaching a binary alone and letting the absence read
as a decision.

## After the tag

### The release notes point at the changelog rather than restating it

Two accounts of one release disagree eventually, and the one on the forge is the
one nobody can correct without editing history somebody has already read.

### The first release closes its own issue and not this one

This checklist is #74. The first release is #75, and it is the first thing this
checklist is followed for. A later release that finds a step here wrong opens an
issue against this file rather than skipping the step, because a step skipped
once is a step that was never in the list.
