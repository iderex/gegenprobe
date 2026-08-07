# 0000. Decision records

Number: 0000
Title: Decision records
Status: accepted
Date: 2026-08-07

## What was decided

Every architecture decision this project rests on is written down before the code
that depends on it exists. The records live under `docs/decisions/`, one file per
decision, named `NNNN-short-slug.md` with a four digit sequence that is never
reused. A number that is retired stays retired; the file stays in the tree.

A record has this shape, and nothing about it is optional:

    # NNNN. Title

    Number: NNNN
    Title: Title
    Status: accepted
    Date: YYYY-MM-DD

    ## What was decided
    ...
    ## Why
    ...
    ## What was rejected
    ...
    ## What this costs
    ...

The first line is a level one heading reading `# NNNN. Title`, with the same
number as the filename and the same title as the header field below it.

The header block is the run of `Field: value` lines that follows the heading,
ending at the first blank line. Four fields are required in every record:
`Number`, `Title`, `Status`, `Date`. `Number` is four digits and equals the
number in the filename. `Date` is `YYYY-MM-DD` and is the date the record
reached its current status. `Status` is one of exactly three values:

- `proposed`, meaning written down and not yet settled
- `accepted`, meaning in force
- `superseded`, meaning replaced by a later record

A fifth field, `Superseded-By`, is required when and only when the status is
`superseded`. Its value is the four digit number of the record that replaced
this one, and that record has to exist.

The body carries four level two sections, in this order, none of them empty:
`What was decided`, `Why`, `What was rejected`, `What this costs`. No other
level two heading appears. Deeper headings inside a section are free.

`docs/decisions/README.md` is the index over those files. It lists every record
under `docs/decisions/` and nothing else, and it is generated rather than typed:

    go run ./tools/decisionindex

Running that on a tree whose index is current leaves no diff. `-check` reports
whether it would change anything and writes nothing, which is the form a gate
leg calls.

A record is never edited to change what it decided. Corrections of spelling, of
a broken link or of a wrong path are edits; a change of substance is not. A
decision that turns out to be wrong gets a new record, whose `What was decided`
names the record it replaces, and the old record's status becomes `superseded`
with a `Superseded-By` pointing forward. Both directions are written, so a
reader who arrives at either end reaches the other.

The machine that refuses a record breaking any of the above is not this record.
It is issue #20, in the scaffolding milestone. Until that lands, everything
written here is a description that nothing enforces, and a malformed record
reaches the default branch exactly as a well formed one does.

## Why

The reason for a decision is worth more than the decision. Six months on, the
question is never what was chosen but whether it would be chosen again, and that
is only answerable if the alternatives and the accepted cost were written down at
the time. A record that lists only the outcome forces the argument to be
reconstructed, and reconstruction is where a decision quietly changes meaning.

The four sections are fixed because the cost section is the one everybody skips.
Making it structural means an empty one is visible, and a record whose cost
section says nothing is a record that has not finished thinking.

Records are append only for the same reason a lab notebook is. An edited record
makes the tree agree with the present, which is the one thing a decision history
must not do. Supersession written in both directions is what replaces editing:
the old reasoning survives, and so does the fact that it was found wanting.

The index is generated because a hand maintained list drifts against the
directory it describes, and it drifts silently. A generated index that is checked
turns that drift into a failure instead of a stale file nobody rereads.

The header block sits below the heading rather than above it so that the file
opens the way a reader expects a document to open, and so that a Markdown tool
reading the first line finds a title there.

## What was rejected

A single `ARCHITECTURE.md` holding everything. It reads well on day one, turns
into a document nobody dares restructure, and it cannot express supersession at
all: replacing a paragraph destroys the thing the register exists to keep.

Recording decisions in issue bodies alone. The tracker is where a decision is
argued, the tree is where it is settled. An operator reading a clone should not
have to reach the network to learn why the harness works the way it does, and an
issue body can be edited without leaving a trace in the artefact.

YAML front matter delimited by `---` for the header. It looks like the case file
format decided in 0002 and is not that format, it needs a parser to read four
fields, and the resemblance would invite somebody to put structure in it.

A free numbering scheme, or dates as identifiers. Sequential four digit numbers
sort, they are short enough to say out loud, and they make a duplicate a thing a
machine can see.

## What this costs

A record per decision is friction on small decisions, and the boundary between a
decision that needs a record and one that does not is a judgement nothing can
make automatically. The bias should be towards writing one.

Append only means the tree accumulates records that are no longer true, and a
reader who searches the directory rather than the index will find them. The
status field is the whole defence against that, which puts weight on the index
and on the status being kept honest.

The format is strict, and strictness in a document is worth nothing until the
check in #20 exists. Between this record landing and that check landing, the
format is a convention held by whoever is writing.
