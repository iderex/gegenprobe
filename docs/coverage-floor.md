# The coverage floor

Floor: 82.2
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

## Raising it

The floor was set from what the suite achieved when it was first measured, and
it is raised deliberately. An aspirational floor is one that gets waived on the
first red run, and a waiver is how a floor becomes decoration.

Raising it is two edits in this file, the number and the date, in a change that
says what made the coverage go up. The leg says when the measured total has risen
far enough above the floor to be worth doing.
