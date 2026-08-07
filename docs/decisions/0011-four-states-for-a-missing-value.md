# 0011. Four states for a value that is not there

Number: 0011
Title: Four states for a value that is not there
Status: accepted
Date: 2026-08-07

## What was decided

Four states, distinct in the schema, in the report and in the words used about
them. Every cell in every artefact this project writes is in exactly one of them.

A cell is never `null`, never an empty string, never a blank, and never zero.
Zero is a physically meaningful energy and a physically meaningful oscillator
strength, so rendering absence as zero is not a shortcut, it is a wrong number.
In the bundle a cell is an object carrying `state` and, for the three absent
states, `reason`. There is no other encoding.

### Measured

The code produced this value and it is here. It carries the value, its
significance count and the significance marker 0008 requires, and no `reason`.

### Declined

The code was asked and did not produce it. This is a statement about a code.

- `not-computed`, the code does not compute that quantity at all.
- `not-in-output`, the code computes it but did not write it in the mode this
  case ran it in.
- `not-converged`, the calculation reached the quantity and did not converge.
- `code-failed`, the code exited unsuccessfully and this value is among what was
  lost.

### Not requested

The case did not ask for this. It is a statement about the question, and it is
not a statement about any code.

- `quantity-not-requested`, the case's property list does not include it.
- `level-not-selected`, the case's configuration or level selection excludes the
  level this cell belongs to.
- `participant-not-in-case`, this participant was not in the run at all.

### Refused

The harness declined to produce or to compare. This is a statement about this
harness.

- `unmatched`, identification returned no match for the level, per 0005.
- `ambiguous`, identification returned more than one candidate and would not
  choose.
- `physics-differs`, the participating codes were not run on the same physics, so
  a comparison between them would be a comparison of two questions.
- `precision-insufficient`, the significance surviving under 0008 does not support
  the comparison being asked for.
- `unit-not-convertible`, no conversion to the canonical unit of 0004 exists for
  what the code reported.
- `run-incomplete`, a limit the harness imposed stopped the step before this
  value existed.

`run-incomplete` is the boundary case and it is placed deliberately. A step the
harness killed at its own timeout or memory limit produces cells that are
`refused`, never `declined`. The code did not fail; it was stopped. Filing that
under `declined` would put a harness configuration decision on a code's record,
which is exactly the confusion the next paragraph forbids.

### The forbidden transitions

No route in the system may turn one state into another. The following are named
because they are the ones a plausible piece of code would do by accident.

- **`declined` to `refused`, and `refused` to `declined`.** This is the pair that
  matters most. One accuses a code of a failure it did not have, the other hides
  a limitation of this bench behind a code's name. Neither is recoverable once
  published, because a reader has no way to see that it happened.
- **Anything to `measured`.** No default value, no fill, no interpolation, no
  carrying a value across from another participant, no substituting a literature
  number. A value that was not produced is not obtainable by any route.
- **`not-requested` to `declined`.** An operator narrowing a case for speed would
  otherwise produce a map that looks like a wall of code failures.
- **`declined` or `refused` to `not-requested`.** The reverse, and worse: it makes
  a failure look like something nobody asked about.

`measured` to an absent state is not a transition, because the states live on
cells and a value has more than one cell. A participant's own result cell can be
`measured` while the comparison cell drawing on it is `refused`, and that is the
normal case for a level identification could not place. Both cells are written,
both keep their own state, and neither overwrites the other. A route that
replaced the result cell instead of writing the comparison cell would be
destroying evidence.

Aggregation follows the same rule. A statistic over a group is `measured` only
where every member of the group is `measured`. Where any member is not, the
statistic takes the state and reason of the strongest absence present, in the
order `refused`, `declined`, `not-requested`, and the count of each is reported
beside it. Nothing is averaged over a partial set and presented as if it were
complete.

### How each one renders without colour

These tables get printed in greyscale and pasted into papers, so every
distinction is carried by characters.

- **Measured** renders as the value and nothing else. No marker, because the
  ordinary case should read as ordinary.
- **Declined** renders as `[DE:not-converged]`, the two letter tag and the
  reason.
- **Not requested** renders as `[NR:level-not-selected]`.
- **Refused** renders as `[RF:ambiguous]`.

Where a column is too narrow for the reason, the tag alone is printed, `[DE]`,
`[NR]` or `[RF]`, and the reasons are listed in a note under the table. The tag
is never dropped, and no width is narrow enough to justify a blank.

Every table carries a legend naming all four states with the count of cells in
that table in each one, including the zeros. Printing the zeros is what stops a
missing legend entry from reading as a missing state, and it puts the coverage of
the table on the same page as the table, which is the same argument 0005 makes
for the headline of the map.

## Why

The entire product of this project is a map of where methods agree and where they
do not. A blank that could mean any of four things is not a weaker signal than a
filled cell, it is an unreadable one, and a reader will resolve it in whichever
direction they already believed.

The distinction between declined and refused is the one that matters most and the
one most likely to be lost, because both look like "no number" to the code that
writes the cell. Declined is a statement about a code. Refused is a statement
about this harness. Publishing one as the other would either accuse a code of a
failure it did not have, or hide a limitation of the bench behind a code's name,
and this project's standing rests on not doing either.

Not requested has to be separate because an operator narrowing a case for speed
would otherwise produce a map that looks like a wall of code failures, and that
map would be shared.

Putting the state in the schema rather than in a note keeps it usable by anything
that aggregates across many maps, which is the main use once more than a handful
of ions have been run. A free text note is readable by a person exactly once.

This mirrors the distinction the readers already have to make when a field is
blank or filled with a fill character, which 0008 already sends here. One
vocabulary rather than two is the point: a reader, the comparison and the
renderer all say the same four words.

## What was rejected

A single missing marker with a free text note. Readable by a person, useless to
anything that aggregates, and the note is the first thing dropped when a table is
reformatted.

Two states, present and absent, with the reasons in a separate log. Puts the
explanation somewhere nobody reads at the moment they need it, which is while
looking at the cell.

Rendering absent values as zero. Common, and here it would be actively dangerous,
because zero is a physically meaningful value for more than one quantity this
project reports.

Rendering absence as an em dash or as a blank with a footnote. Compact, and it
collapses the four distinctions this record exists to keep apart.

Folding `run-incomplete` into `declined` because the code did in fact stop
running. Simpler to implement, and it files a harness limit under a code's name.

Distinguishing the states by colour. It reads well on a screen and it survives
neither a printer nor a reader who cannot distinguish the colours chosen.

## What this costs

Four states have to be threaded through the model, the readers, the comparison,
the schema and the renderer, and every new quantity has to answer the question
for itself. There is no place in this design where absence can be handled once.

The report is busier than one with a single blank. A table where most cells are
tagged is hard to read, and the honest answer to that is that such a table is a
hard result, not a rendering problem.

Legends with zero counts add lines to every table, including the many tables
where nothing is absent. That is a deliberate cost paid so that a reader learns
the vocabulary from the first table they see rather than from the first table
that needed it.

The `refused` reason list will grow. Every new way the harness can decline is a
new entry, and an implementation that meets an unlisted case and picks the
nearest reason is doing the thing this record forbids. The correct handling is to
add the reason and supersede this record.

Nothing here is enforced today. No check in this tree refuses a cell written as
`null`, a state changed in flight, or a statistic aggregated over a partial
group. The schema work in the reading milestone and the architecture conformance
work in the quality milestone are where those would be refused, and until they
land this record is a description.
