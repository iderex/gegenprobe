# 0006. What the harness means by agreement, and what it refuses to conclude

Number: 0006
Title: What the harness means by agreement, and what it refuses to conclude
Status: accepted
Date: 2026-08-07

## What was decided

Agreement is never reported as a boolean, and the harness never declares a
winner.

For each matched level and each matched transition, the map reports the actual
values from every code that produced one, the spread across them, and the
identity of the extremes. Where the operator declared a tolerance in the case
file, the map additionally reports whether the spread falls inside it, and it
labels that as the operator's tolerance rather than as a verdict.

The harness does not compute a recommended value and does not average across
codes. It does not rank the codes. It does not describe a disagreement as an
error in any code.

Where the codes were not run on the same physics, because a per code knob changed
something shared, the comparison is refused for the affected quantities rather
than reported with a caveat.

A disagreement is the expected output. The report says so in its own words, in
the place a reader arrives first.

### Every statistic the map reports, and what it does not license

For a matched quantity, and nothing beyond this list:

- **The values.** One per participating code, each carried in the canonical unit
  and alongside the code's own printed value, per 0004, and each printed to the
  significance the code produced, per 0008. This does not license reading a
  difference in the last printed digit as a difference in the physics; two values
  printed to different significance were never comparable at the finer one.
- **The count of codes that produced a value**, and the count that did not, split
  by the reason vocabulary in 0011. This does not license reading a missing value
  as dissent. A code that was not asked, could not represent the quantity, or
  failed is silent, and silence is not disagreement.
- **The minimum and the maximum, each with the code that produced it.** This does
  not license calling either code wrong, and it does not license calling the
  extreme an outlier. Two codes at the ends of a range of three are the ends of a
  range of three.
- **The absolute spread**, maximum minus minimum, in the canonical unit. This is
  not an uncertainty on the quantity and it is not a standard error. The codes
  are not independent samples of anything; several of them share ancestry,
  approximations and in places whole subroutines, so no sampling statistic
  applies to them.
- **The relative spread**, the absolute spread divided by the midrange of the
  values. The midrange is a scale for reading the spread and it is not an
  estimate of the quantity. It is not a recommended value, it does not become one
  by being the only number in the row that looks like an answer, and the map
  labels it as a scale everywhere it appears.
- **Whether the spread falls inside the operator's declared tolerance**, present
  only where the case file declared one, and named as the operator's tolerance
  wherever it is shown. This does not license reading it as a verdict on the
  codes. It says the operator's own threshold was or was not met, and the
  threshold came from the case file, not from this project.

No mean, no median, no weighted combination, no standard deviation, no score, no
ordering of the codes by anything.

### The refusal rule for mismatched physics, and how it is detected

A comparison is refused for a quantity when the codes that produced it were not
asked the same question about it.

Detection rests on two sources, and both are read.

The first is the case, before anything runs. The shared physics block of a case
is by construction identical for every code, since there is one of it. What can
differ is the per code section. Every per code knob in the case schema is marked,
in the schema and with no default, as either presentational or physics affecting,
and a physics affecting knob declares which quantities it reaches. A knob that
declares no reach is treated as reaching all of them. Where the effective
physics affecting inputs differ between two codes in a run, the quantities those
knobs reach are refused for that run.

The second is what the run actually did, after it ran. Each code's run record
states the model it really built rather than the one it was asked for: the
expanded configuration set, the size of the resulting basis, and which physical
effects were included, for instance whether the Breit interaction or a QED
correction was on. Where those recorded facts differ in a physics affecting way
between two codes, the affected quantities are refused even though the case
declared nothing. This is the path that catches a code silently defaulting to
something the others did not do, which is the case the first source cannot see.

A refusal is a first class entry in the map. It names the quantity, the codes
involved, the input or the recorded fact that differed, and which of the two
sources detected it. It is not a footnote, it is not a warning, and it does not
carry a number beside it that a reader could copy.

Where a refusal covers only some quantities, the others are compared normally.
Refusing the whole run because one knob differed would push operators towards
setting nothing, which is the opposite of what the rule is for.

## Why

The scatter across a handful of codes is currently used as an uncertainty
estimate in published work. That practice is what this project is built to make
systematic, and it would be undone by a tool that collapses the scatter into a
single number, because the single number would immediately be cited as the
answer.

Declaring a winner requires a weighting across methods that nobody has agreed on,
and the disagreements this bench will find are precisely the cases where such a
weighting is least defensible.

Refusing rather than caveating, where the physics differed, is the stricter
choice and the correct one. A caveat gets dropped when the number is copied into
a table. A missing number gets asked about.

Calling a disagreement an error in one code would be a claim about which one is
right, made by a program that cannot know. It would also make groups reluctant to
have their code included, which would shrink the bench to the codes whose authors
are most relaxed and destroy the comparison.

Saying plainly that disagreement is the expected output is not a soft statement.
A tool whose interesting result reads as a failure gets its interesting results
suppressed by its own users.

Reading the run record as well as the case is what keeps the refusal rule from
being a formality. A rule that only inspects what the operator wrote catches the
operator's mistakes and none of the codes' defaults, and the defaults are where
the silent mismatches live.

## What was rejected

A pass or fail verdict per level against a fixed tolerance. Immediately readable,
and it hard codes a judgement about acceptable accuracy that varies by ion, by
quantity and by application.

A recommended value with an uncertainty, computed from the spread. This is the
thing the field would most like to have and the thing this project is least
entitled to produce, at least until the weighting question is settled by somebody
other than the tool.

Ranking the codes by agreement with an experimental reference. A legitimate
separate product, and it turns a comparison bench into a scoreboard, which
changes who is willing to participate.

Normalising the relative spread by the median rather than by the midrange. The
median of three or four values is one of the values, which makes it read as a
chosen answer, and choosing an answer is the thing this record forbids.

Reporting a standard deviation because it is cheap to compute. It would import a
sampling model that does not hold, into the one number readers trust most.

Refusing the entire comparison when any physics affecting input differs. Safe,
and it teaches operators to leave the per code section empty, which removes the
information the detection depends on.

## What this costs

The primary output is harder to read than a single number, and some users will
want the single number badly enough to compute it themselves from the bundle.
That is fine, and it means the choice of weighting is theirs and is visible.

Refusing comparisons where the physics differed will produce empty regions in
maps that a caveat would have filled, and those regions need explaining every
time.

Requiring every per code knob to declare its reach is a real burden on whoever
adds a code, and getting a declaration wrong produces a comparison that should
have been refused. Nothing here can check that declaration; it is a judgement,
and the review of a new participant is where it is caught.

The second detection source only works to the extent that a code reports what it
actually did, and the codes differ widely in how much they report. Where a code
says little, the harness will pass comparisons it would have refused had it
known, and the honest form of that limitation is to say per participant what its
run record covers rather than to imply the check is complete.
