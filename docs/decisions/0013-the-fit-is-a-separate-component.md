# 0013. The fit component inherits no language from the harness

Number: 0013
Title: The fit component inherits no language from the harness
Status: accepted
Date: 2026-08-07

## What was decided

The parameter fit is a separate component, started by the harness the way a code
is started, behind the same process contract. It ships after the harness and it
is not in the first release.

It inherits nothing from 0001. When its milestone begins, the language and the
numerical library are chosen again from that problem's own requirements, the
choice is argued from the standpoint the same way every other means choice is,
and it becomes its own record. This record fixes that the question is asked, not
what the answer is.

Two things are fixed now, because work that lands before the fit depends on them.

### The process contract

The fit is held to the contract a code is held to, in the same terms, so that the
runner has one shape and not two.

- It runs in a container built from a recipe in this repository, by the rules of
  0003. It is not a library linked into the harness and it is not a subprocess
  reaching into the harness's memory.
- Its image carries a baked manifest with the same fields 0003 names for a code:
  the component identity and version, the source checksum, the patch list, the
  base image digest, the compiler and its version, the compile and link flags,
  the numerical libraries with their versions, the platform, and the recipe
  revision. A fit result whose provenance is thinner than a code's would be the
  one number in the bundle nobody could reproduce.
- It is started rootless and it never asks for elevation, on the same terms as
  every other step.
- It receives exactly one input: a canonical JSON document, on a path the runner
  fixes, in the form 0002 defines for canonicalisation. Nothing else is passed in.
  No network, no arguments carrying physics, and no environment beyond the
  allowlist the run record captures.
- It writes exactly one output: a canonical JSON document, on a path the runner
  fixes, in the common model of 0004. The harness reads that and nothing else. It
  does not parse the component's log, and the log goes to the raw area outside
  the checksummed bundle, as a code's output does under 0007.
- A quantity it did not produce is absent under 0011, with a reason from the
  `declined` list, and the ordinary one here is `not-converged`. The fit not
  converging is a result, not an error.
- The harness owns the timeout and the resource limits. Where one fires, the step
  is stopped and the affected cells are `refused` with `run-incomplete`, which is
  the same handling a code gets, for the same reason: a limit this project set is
  not a failure of the thing it was set on.
- It is deterministic given its input, or it declares itself otherwise under 0008.
  An optimiser with a stochastic element takes its seed from the input document,
  never from the clock or from the operating system, and records the seed it used
  in its output. A seed the input did not carry is a run nobody can repeat.

### The schema obligation for observed level energies

The common data model carries observed level energies alongside calculated ones,
from the model milestone onward, whether or not anything reads them yet.

A level may carry zero or more `observed` entries. Zero or more, not zero or one:
two compilations can disagree about the same level, and a schema that admits one
value forces somebody to choose between them silently.

Each entry carries the following, and a consumer may rely on all of it.

- `value`, the energy in the canonical unit of 0004, as decimal text, on the same
  terms as every other number in the model.
- `significance` and its marker, exactly as 0008 requires of a calculated value.
  A published value with four digits is a four digit value.
- `uncertainty`, an object with `value` in the same unit and as decimal text, and
  `kind`, one of `standard`, `expanded`, `stated-interval` or `unstated`. Where
  `kind` is `expanded` the entry carries `coverage-factor`. Where the source
  printed no uncertainty, `kind` is `unstated` and `value` is absent under 0011
  with reason `not-in-output`; it is never filled with a guess and never with
  zero.
- `source`, an object with `citation` as the source printed it, `identifier`
  holding a digital object identifier, a bibliographic code, or a database name
  with its version, and `retrieved` as a `YYYY-MM-DD` date. A source without a
  resolvable identifier is admissible and says so; a source with none at all is
  not.
- `label-as-published`, the designation the source gave the level, kept verbatim
  as text. It is carried so a reader can find the row in the original, and it is
  never parsed, never normalised and never used to attach anything to anything.
  0005 forbids matching on a printed label and this field is not an exception to
  it.
- `association`, saying how this entry came to be attached to this level, one of
  `by-operator` where the case named the pairing, or `by-source-identifier` where
  the source keys on the same level identity the case used. There is no third
  value. In particular there is no association by energy proximity, because
  attaching an observation to the nearest calculated level and then reporting the
  distance between them measures nothing.

Observed values are a separate column and never a participant. They do not enter
the code-to-code statistics, they are not a reference the codes are scored
against, and they cannot become one without superseding 0006. What they are for
is giving the fit something to fit to, and giving a reader something to look at
beside the spread.

### The success criterion for the fit

The first criterion is agreement with what the existing interactive procedure
produces, on cases where that procedure has a published result. Before anything
else, and before any claim about quality.

An optimiser that finds a better minimum than the published fit is an interesting
result that has to be argued for. It is not a passing test, and a test suite that
treated it as one would be unable to tell a discovery from a bug.

## Why

The fit is an optimisation problem with many local minima, and the reason it is
done interactively today is that a person steers it away from the wrong ones.
Replacing that with an automatic procedure is a research task rather than a port.
Giving it its own component keeps its uncertainty out of a harness that has to be
boringly reliable, and keeps a long running numerical process out of the address
space of the thing that has to survive a code segfaulting next to it.

Deciding now that it does not inherit the harness language is the whole point of
writing this early. The harness is Go for reasons that are about process control
and distribution, and none of those reasons apply to nonlinear least squares over
a rough landscape. Left unstated, the language would be inherited by default and
the decision would never be made at all, which is the failure mode a means check
exists to prevent.

Holding the fit to the code contract rather than to a friendlier one is what
keeps the runner single. A second contract would mean a second timeout
implementation, a second manifest shape, a second way of recording a failure, and
two places for the absent-value vocabulary to drift.

The schema obligation is here rather than in the fit milestone because
retrofitting observed values into a model that only ever held calculated ones is
the kind of change that ripples through every reader and every bundle written up
to that point. Discovering it at the fit milestone would mean either a schema
break or a second place where energies live.

Admitting several observed entries per level, each with its own source, is the
same argument 0011 makes about absence. Collapsing two disagreeing compilations
into one number is a judgement, and a judgement made silently inside a schema is
the worst place for one.

Making agreement with the existing procedure the first criterion is a deliberate
choice of a boring target over an ambitious one. A fit that disagrees with the
published one is either a discovery or a bug, and there is no way to tell them
apart without first showing the tool reproduces what is already known.

## What was rejected

Building the fit inside the harness binary. Fewer moving parts, one distribution,
and it would put a numerical research problem in the same process as the part
that has to be dependable, sharing its crashes and its memory.

Deferring the whole question until the harness ships. Cheaper now, and it leads
to a data model with no place for observed values, discovered at the worst
moment.

Letting the fit inherit Go by default and revisiting later. The revisit does not
happen. A means carried over from the last artefact is an assumption about this
one, and it is the assumption this record refuses.

Making a better minimum the headline goal. It is the eventual goal and it is the
wrong first target, since nothing can be claimed about a better minimum until
reproduction of the known one is demonstrated.

A single observed value per level, chosen by the harness from the available
sources. Simpler schema, and it makes the project the arbiter of which
compilation is right, which is a claim it has no basis for.

Attaching observations to levels by energy proximity. It is what would be reached
for first, it produces a complete looking column, and it makes the residual a
function of the attachment rather than of the physics.

## What this costs

A second component with its own toolchain, its own tests and its own maintenance,
in a project that has argued hard for keeping the dependency surface small. That
cost is accepted because the alternative is worse in a way that is harder to
undo.

Carrying observed values in the model before anything uses them means schema
fields that sit empty through the first release and have to be explained to
everybody reading it. A field that is always absent invites somebody to remove
it, and this record is the answer to that when it comes up.

Holding the fit to the container contract means the fit cannot be run as a quick
script during its own development without building an image, which is friction on
exactly the component whose development is most exploratory. The answer is that
the contract binds what the harness starts, not what a researcher does on their
own machine, and only what the harness starts produces a bundle.

The `association` field with no proximity value means some legitimate pairings
cannot be expressed at all until an operator writes them down by hand. That is
work, it is the correct work, and it will be experienced as the schema being
awkward.

Nothing here is enforced. The fit component does not exist, the model milestone
has not landed, and no check in this tree refuses a model without an `observed`
block or a component started outside the contract. This record is what the model
milestone is required to implement, and until it does, it is a description.
