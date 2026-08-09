# The coverage floor

Floor: 80.4
Last raised: 2026-08-08

The two fields above are read by the `coverage` leg of `go run ./gate`, which
refuses a run whose measured total is below the floor. They are here rather than
as a constant in the gate's source so that lowering the floor is a change to a
document somebody reviews rather than a number somebody adjusts while making a
red run go green.

## What the number covers, and what it does not

The floor and the measurement are over the gate tier, which is the tier record
0009 defines as the one carrying no build tag: unit and contract tests over
fixtures committed to this repository.

The parts of this project that most need testing are not in that tier and
contribute nothing to the number. The runner against a real container engine,
the recipes against real codes, and a case run end to end are the integration
harness and the physics regression, under the `integration` and `regression`
tags, and neither is measured here. A high figure is a statement about the
readers, the model, the comparison and the gate itself. It is not a statement
about the tool.

That sentence is repeated by the leg every time it prints the number, in the same
place as the number, because a caveat that lives only in a document is a caveat
nobody reads next to the figure they are about to quote.

## Why it has moved

82.2 to 81.1, on 2026-08-08, when the `dependencies` leg landed.

That leg has to run `go mod verify` and a build, so part of it is statements only
a process reaches. Record 0009 refuses `os/exec` in this tier, so those
statements cannot be covered by the suite that measures them, and every leg of
this kind that arrives lowers the percentage while covering everything a test can
reach. The judging in that leg is at 100%; what is not covered is the shelling
out.

That is a reason about the tree rather than a number adjusted to match a run, and
the difference is the whole point of this file. A lowering with no section here is
the move this document exists to make visible.

81.1 to 80.4, on 2026-08-09, when the `commit hygiene` leg landed. It is the same
reason again and it is worth writing rather than pointing at. That leg reads git,
so resolving a range, asking for the log and turning a failure to read history
into a verdict are statements only a process reaches, and record 0009 refuses
`os/exec` in this tier for the tests that would reach them. What can be covered
is covered: the judging in `internal/commit` is at 100% of statements, over
fixtures holding recorded ranges, and what is not is the shelling out and the two
thin callers around it.

The `Last raised` field above is unchanged, because this is not a raise. That
date records when the floor last went up and the leg quotes it beside the number,
so moving it on a lowering would turn that sentence into one saying the opposite
of what happened.

Two lowerings in two days is a pattern rather than an incident, and it is the one
this file predicts: every leg that has to run something lowers the percentage
while covering everything a test in this tier can reach. What the number is
turning into is a floor over the judging with the process boundary of each leg
subtracted from it, and at some point the honest repair is to measure the two
separately rather than to keep taking a point off a single figure. That is not
done here, and this paragraph is the record that it is owed.

## Raising it

The floor was set from what the suite achieved when it was first measured, and
it is raised deliberately. An aspirational floor is one that gets waived on the
first red run, and a waiver is how a floor becomes decoration.

Raising it is two edits in this file, the number and the date, in a change that
says what made the coverage go up. The leg says when the measured total has risen
far enough above the floor to be worth doing.
