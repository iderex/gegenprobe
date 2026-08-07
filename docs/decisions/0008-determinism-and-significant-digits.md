# 0008. Determinism in the harness, and never printing more digits than a code produced

Number: 0008
Title: Determinism in the harness, and never printing more digits than a code produced
Status: accepted
Date: 2026-08-07

## What was decided

The harness is deterministic given its inputs. Sorted iteration everywhere that
output is produced, no reliance on map ordering, no parallelism that can reorder
results, no wall clock or hostname anywhere inside compared data.

The codes are not assumed to be deterministic. Where a code is known to produce
run to run variation, that is recorded as a property of the code, and the harness
reports it rather than smoothing it.

Numbers are printed with the precision the code produced and never more. Each
reader records, per quantity, how many significant digits the code actually
wrote, and every derived quantity is printed to the smallest of the precisions
that fed it. Where a value has to be converted between units, the conversion does
not add digits.

Comparisons between floating point values use declared absolute and relative
tolerances that are stated in the artefact, not compiled in silently.

### The significance rule, per quantity type

Every value in the model carries an integer count of significant digits and a
marker saying how the reader arrived at it: `stated` where the code documented
its own output precision, `counted` where the reader counted digits in the field,
and `format` where the reader took it from the code's fixed format specification.

- **A number in scientific notation**, such as `1.2345E+03`. The significance is
  the number of digits in the mantissa. This is the unambiguous case.
- **A number with a decimal point**, such as `1234567.89`. The significance is
  the count of digits from the leading non zero digit to the last printed digit,
  trailing zeros after the point included. A trailing zero after the point was
  printed because the format printed it, so it is significant.
- **An integer with no decimal point and trailing zeros**, such as `109700`. The
  significance is ambiguous and the reader does not guess. It records the count
  as an upper bound, sets the `trailing-zero-ambiguous` marker, and every
  comparison involving that value reports the marker. This is rare in fixed
  format output and it is not rare enough to leave to a convention.
- **A field that is blank, or filled with a fill character**, is not a value at
  all. It goes to the absent vocabulary in 0011 and carries no significance.
- **An exact integer label**: a level index, the numerator of a half integer J,
  the nuclear charge, a multipole order. These are exact, carry no significance
  count, and are never printed in a form that suggests one.
- **A quantity a code prints in a documented fixed width**, where the code's own
  documentation states the precision it computes to, takes the documented number
  and the `stated` marker, even where the field would allow more digits. A code
  padding a four digit result into a twelve character field has produced four
  digits.

The same rules apply to level energies, transition energies, wavelengths,
weighted oscillator strengths, transition probabilities, line strengths and
mixing weights. There is no per quantity exception, and where a reader has to
depart from them for a particular code, the departure is stated in that reader's
own documentation with the reason, per participant.

**A value that came out of a unit conversion** keeps the significance of the
value it came from, and is printed to that many significant digits. The
conversion factor never adds digits, and this holds whether the factor is exact
under the SI or measured. Where the factor is measured and its own relative
uncertainty is larger than half a unit in the last digit the source value would
otherwise keep, the significance of the result is reduced until that stops being
true, and the reduction is recorded on the value so a reader can see that the
constant, not the code, set the limit. This will almost never bite at the
precision these codes print, and the rule exists so that the almost is not a
silence.

**A derived quantity** is printed to the smallest of the significances that fed
it, with one refinement that matters more than the rule it refines. Where the
derivation is a difference of two nearly equal values, a transition energy taken
from two level energies being the ordinary case, digit counting overstates what
survives. The absolute precision of each input is half a unit in its last
significant digit; the absolute precision of the difference is the larger of the
two; and the significance of the result is the number of digits from its own
leading digit down to that absolute precision. Two energies of eight significant
digits differing in the fifth leave a difference with four, and printing eight
would be inventing four. Cancellation is the case this project will meet on every
transition, so it is the case the rule is written for.

### Where tolerances live, and what the defaults are

Two different things are called a tolerance here and they are kept apart.

**The operator's agreement tolerance** lives in the case file, in an optional
`tolerances` block, keyed by quantity type. Each entry may carry an `absolute`
value in the canonical unit for that quantity, a `relative` value that is
dimensionless, or both. A spread is inside tolerance when it satisfies every
bound that was declared for that quantity. There is no default: a quantity with
no declared tolerance gets no tolerance judgement at all in the map, and the map
says so rather than showing a blank that reads as a pass. What this tolerance
does and does not license is 0006 and is not restated here.

**The harness's own numeric tolerance** is what it uses when it has to decide
whether two floating point numbers are the same. Most of the time it does not
have to: canonical values are carried as decimal text, per 0002 and 0004, so
equality is exact and no tolerance is involved. Where a comparison genuinely
needs one, checking that a conversion round trips or that a regression matches a
recorded result, the default is a relative tolerance of `1e-9` and an absolute
tolerance of zero.

Both kinds are written into the bundle's run record, every tolerance that was in
force during the run, each marked `declared` or `default`. A reader of the
artefact can see what standard was applied without reading the source, which is
the whole point, and a default that is written down is no longer a silent one.

## Why

Printing more digits than a code produced is the cheapest way to invent a
disagreement. Two codes that both wrote four significant figures, compared at
fifteen, differ in the last eleven for no physical reason, and a map full of such
differences would be worse than useless because it would look like data.

Determinism in the harness is what makes the bundle diffable, which is the
decision taken in 0007 on the bundle and the report being separate artefacts. A
single unsorted iteration is enough to destroy that property, and it will not be
noticed on a small case.

Treating code level nondeterminism as a recorded property rather than as noise to
be averaged away follows from what this project is for. If a code gives different
answers on two runs of the same input, that is a finding about the code, and this
project's own plan says a finding of that kind is the expected output rather than
an embarrassment.

Tolerances belong in the artefact because a reader has to be able to see what
standard was applied without reading the source.

The cancellation refinement is not a nicety. Transition energies are the quantity
this bench will report most often, they are differences of large nearly equal
numbers, and a digit counting rule applied naively would print a transition
energy to the significance of the level energies it came from. That single
mistake would put invented digits on the project's most cited output.

## What was rejected

Printing full double precision everywhere and leaving significance to the reader.
Honest in intent, and it produces tables that no reader will correct.

Rounding everything to a fixed number of digits chosen once. Simple, and wrong in
both directions: too coarse for a code that gives ten digits, too fine for one
that gives three.

Averaging repeated runs of a nondeterministic code. Statistically reasonable and
it hides exactly the property worth reporting.

Guessing at the significance of a trailing zero integer by convention, either by
counting it or by discounting it. Both conventions exist in the literature, both
are wrong about half the time, and a recorded ambiguity is the only answer that
does not manufacture information.

Giving the operator's agreement tolerance a default. Any default would be a
judgement about acceptable accuracy made by this project, which is precisely what
0006 refuses, and it would be applied to every case whose author never thought
about it.

## What this costs

Every reader now has an extra obligation: it has to record how many digits it
saw, not just what the number was. That is fiddly for fixed format output where a
trailing zero may or may not be significant, and the readers have to state the
rule they applied per field.

Refusing parallelism that could reorder results costs wall clock time on cases
with many codes. Where parallelism is safe, it is allowed, and the ordering is
imposed at the point of writing rather than by the order things finished.

The cancellation rule will produce transition energies with startlingly few
significant digits, and that will be read as a defect in the harness before it is
read as a property of the inputs. The report has to explain it in the place the
number appears, every time, or it will be worked around.

Carrying significance and its provenance marker on every value widens the schema
and the bundle further than 0004 already widens it, and most rows will carry the
same two fields with the same two values.
