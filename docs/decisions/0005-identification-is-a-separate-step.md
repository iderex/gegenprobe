# 0005. Identifying the same level across codes is a separate step that is allowed to fail

Number: 0005
Title: Identifying the same level across codes is a separate step that is allowed to fail
Status: accepted
Date: 2026-08-07

## What was decided

Identification runs before comparison, has its own output in the bundle, and is
never a side effect of comparing. Nothing in the comparison step may create,
alter or discard a match.

Its result is a set of match groups over the levels the codes reported, plus two
result classes that are outcomes rather than errors: unmatched and ambiguous.
Both are counted in the headline of the agreement map, beside the number of
levels compared, and neither appears only in a footnote.

The default matcher uses only properties that are not a matter of one code's
notation. It never reads a code's own spectroscopic label.

### The matching rule

The rule below is stated so it can be implemented without a second reading of
this record. Where it names a threshold, the threshold is a declared parameter
with a default, the case may set it, and the value in force is written into the
bundle beside the result.

**Blocks.** Every level is placed in a block keyed by total angular momentum `J`
and parity. Matching happens inside a block and never across one. A code that did
not report `J` or parity for a level does not get that level matched: it is
unmatched with the reason `symmetry not reported`, and it is not repaired by
inference from anything else.

**Canonical configuration.** Each level's leading configuration is reduced to a
canonical form before anything compares it: an ordered list of `(n, l,
occupancy)` for the open subshells, with closed subshells common to every
participant dropped, and with the code's coupling label, its term symbol and its
seniority discarded. Two configurations are equal when their canonical forms are
equal. Nothing else counts as equality, and in particular no string comparison of
a printed designation counts.

**The gate.** A pair of levels from two codes is a candidate only if all of the
following hold. `J` is equal. Parity is equal. The canonical leading
configurations are equal. The leading mixing weight is at or above the dominance
threshold `w_min` in both codes. Any pair failing any of these is not a candidate
and cannot become one later.

`w_min` defaults to 0.5, which is the point at which the leading component is a
majority of the eigenvector rather than merely the largest piece of it. It is a
convention and not a measurement, and the example cases in the case
specification milestone are where a different default would be argued from.

**Scoring inside a block.** Candidates are scored on two quantities, in this
order and never the reverse. First, the difference in leading mixing weight
between the two levels. Second, the difference in ordinal position within the
block, where position is the level's rank by energy among that code's levels in
that block. Energy ordering enters only here, after the invariant properties have
already agreed, because ordering is precisely what the methods disagree about.
Absolute energy differences are not used at all: they are the quantity being
measured, and using them to decide what to measure would build the answer out of
itself.

**The assignment.** Within a block the matcher computes a one-to-one assignment
over candidates that minimises the total score. It is an assignment and not a
nearest neighbour sweep, so no level is claimed twice and the outcome does not
depend on the order the levels were read in. A level left without a partner is
unmatched.

**Confident.** A match is confident when it is in the assignment, when the pair
passed the gate, and when the best alternative partner for either member scores
worse by at least the separation margin `m_min`. `m_min` defaults to one ordinal
position, meaning the runner up is at least one rank further away; a case may
raise it. A confident match carries the reason it was proposed, which is the list
of properties that agreed and the margin that separated it.

**Ambiguous.** Where two or more candidates are within `m_min` of each other and
the matcher cannot separate them, the result is ambiguous. It is not resolved by
a coin flip, by input order, or by taking the closest energy. Every candidate is
named in the result, the levels involved are not compared, and the group is
counted as ambiguous in the map.

**Unmatched.** A level is unmatched when no candidate survives the gate, when its
leading weight is below `w_min` in either code so its leading configuration is
not a property of the state, when its block is empty in the other code, or when
it is the leftover of an assignment. The reason is recorded from that list, one
value, never a generic blank.

**Three codes and more.** Identification is pairwise against a reference
participant named in the case, and a group is confident only where every pairwise
match in it is confident. One ambiguous or unmatched edge makes the whole group
ambiguous or unmatched respectively, because a group is a claim that all its
members are the same level and a claim is not partly true.

**The operator's mapping.** A case may carry an explicit mapping. Where it does,
it overrides the matcher for the levels it names, the affected rows carry a mark
saying so wherever they are rendered or serialised, and the mark travels into the
bundle and into every report drawn from it. It is not a confidence value and it
does not raise one; it is a statement that a person decided this row.

### The worked case where the naive rule is wrong and this one declines

Neon-like iron, Fe XVII, in the block of odd parity levels with `J = 1` arising
from the `n = 3` complex.

Two configurations reach that block. `2p^5 3s` is odd and its terms are `1P` and
`3P`, giving two levels with `J = 1`. `2p^5 3d` is odd and its terms are `1P`,
`1D`, `1F`, `3P`, `3D` and `3F`, giving three levels with `J = 1`, conventionally
written `1P1`, `3P1` and `3D1`. `2p^5 3p` is even and does not reach this block.
So the block holds five levels, three of which share one leading configuration.
That structure follows from angular momentum coupling and is derived here rather
than measured; no code has been run in this tree, and this record quotes no
energies and no orderings.

A code working in jj coupling describes the same three `2p^5 3d` levels as
`(1/2, 3/2)_1`, `(3/2, 3/2)_1` and `(3/2, 5/2)_1`, where the pair is the hole's
`j` and the `3d` electron's `j`. A code working in LS coupling describes them as
`1P1`, `3P1` and `3D1`. The two alphabets do not map onto each other by string
comparison, and the mapping between them is not fixed: it is a function of the
mixing, which is exactly the thing the two methods disagree about. A code that
prints an LS designation for a state that is not an LS eigenstate is printing the
name of its largest component, so the label moves when the mixing moves.

The naive rule, matching on the code's own designation, therefore does one of two
wrong things here. Between codes in different coupling schemes it matches nothing
and reports a total failure that is an artefact of notation. Between two codes
that both print LS designations it matches `3D1` to `3D1` and calls it done,
which silently asserts that the two codes agree about the composition of a state
whose composition is the open question. A wrong pairing there shows up downstream
as a large method disagreement, which is exactly the signal this project exists
to publish.

The rule in this record reaches the block, agrees on `J` and parity for all five
levels, separates the two `3s` levels from the three `3d` levels on canonical
leading configuration, and then has three levels sharing one leading
configuration and one property left. Where the leading weights are below `w_min`
it declines: those levels are unmatched with the reason that the leading
configuration is not dominant. Where the weights clear `w_min` but two candidates
sit within `m_min` of each other, it returns ambiguous and names both. In neither
case does it produce a row, and in both cases the count appears in the headline
of the map.

## Why

Labels differ between codes by construction, and near degenerate levels change
order between methods. Any matcher that always produces a full mapping is
guessing somewhere, and the guesses are invisible in the artefact.

The failure mode this guards against is specific and severe. A wrong match shows
up as a large method disagreement, which is the most valuable output this project
has. A bench whose worst failure is indistinguishable from its best result cannot
be trusted at all, so non-identification has to be a first class result rather
than an internal detail that a log records.

Making the unmatched and ambiguous counts part of the headline is the other half
of the same argument. A map covering sixty per cent of the levels and a map
covering all of them are different claims about the same ion, and a reader who
has to dig for that number will not dig.

Ordering within a block is used only after the invariant properties agree,
because ordering is what the methods disagree about. Using it first would build
the answer out of the thing being measured. Absolute energy differences are
excluded from scoring entirely for the same reason, one step further.

Separating identification from comparison is what makes both of them testable.
Identification is a function from two level lists to a set of groups, comparison
is a function from groups to statistics, and each can be given fixtures and a
known answer. Fused together they can only be tested end to end, which is where a
confident wrong answer hides.

## What was rejected

Matching on the code's own spectroscopic label. The cheapest rule available, and
it fails wherever two codes use different coupling schemes for the same level,
which is the normal case in the heavy complex systems this project cares about.
The worked case above is the shape of that failure.

Matching purely by energy ordering within a symmetry block. Simple, defensible
for light systems where levels are well separated, and it silently swaps near
degenerate pairs, which is the case that matters. It also makes the answer depend
on the measurement.

Matching against an external reference such as a level database, and comparing
each code to that. That measures each code against observation, which is a
legitimate and different product, and it makes the bench unusable for the ions
where no such reference exists, which are the ones worth running.

Forcing a total mapping with a nearest neighbour rule. Produces a complete
looking map, needs no thresholds, and hides its own failures behind full coverage.

A confidence that is a continuous score with no refusal. It reads as more
informative and it moves the decision to the reader, who will take the highest
number available rather than none.

## What this costs

The map has holes, and their size is not knowable in advance. That has to be
explained wherever the map is presented, or a reader will read a hole as
agreement. Putting the counts in the headline is the defence and it is not a
complete one.

The confidence classification rests on two thresholds that are conventions rather
than measurements. They will be argued about, and that is preferable to the same
judgement being made silently inside a scoring function.

Requiring a mixing weight means every reader has to extract one, and not every
code prints it in every mode. Where a reader cannot supply it, every level it
produced falls below `w_min` by construction and the whole participant goes
unmatched. That is a severe outcome from a missing field, it is the correct one,
and it puts a hard requirement on the reader contract in the reading milestone.

Supporting an operator supplied mapping adds a route by which a determined user
can force the answer they expected. Marking every affected row is what keeps that
honest, and marking does not prevent it.

Pairwise matching against a reference participant makes the choice of reference
visible in the result, and a group that is confident against one reference can be
ambiguous against another. The alternative, a simultaneous assignment over all
participants, hides the same instability inside one number.

Nothing here is enforced today. No check in this tree refuses a comparison drawn
from a non-confident match, and the route that would refuse it is the comparison
milestone's own work. Until that exists this record is a description of what the
implementation is required to do.
