# 0004. The common data model, and the reciprocal centimetre as the canonical unit

Number: 0004
Title: The common data model, and the reciprocal centimetre as the canonical unit
Status: accepted
Date: 2026-08-07

## What was decided

One versioned schema with three parts: a run record, a table of levels, and a
table of transitions.

A level carries the code's own index, the code's own label kept verbatim and
unparsed, the quantum numbers the code actually stated (total J, parity, and
where available the leading configuration with its mixing weight), and the
energy. The energy is stored twice: once exactly as the code printed it, with the
unit the code used, and once converted to the canonical unit.

The canonical unit for level energies is the reciprocal centimetre. Wavelengths
are stored in nanometres in vacuum, with the code's own value and unit kept
alongside in the same way. Transition strengths are stored as the weighted
oscillator strength and the transition probability, each with the multipole type
and order that produced them.

A transition refers to levels by their identity in this model and never by a
code's own index alone.

Every field that a given code does not produce is absent and distinguishable from
zero. The vocabulary for that is decided separately, in 0011.

### The canonical unit for every quantity the model carries

| Quantity | Canonical unit | Conversion from what codes print |
| --- | --- | --- |
| Level energy | cm^-1 | from eV, hartree, rydberg or cm^-1, below |
| Transition energy | cm^-1 | difference of two canonical level energies, or the same conversions where the code printed it directly |
| Wavelength | nm, in vacuum | `lambda_nm = 1e7 / sigma_cm^-1`, exact by definition of the two units |
| Weighted oscillator strength gf | dimensionless | none; `gf` is carried as printed and `f` is multiplied by the lower level's statistical weight `g = 2J+1` where a code prints `f` alone |
| Transition probability A | s^-1 | from the atomic unit of time where a code prints in atomic units |
| Line strength S | atomic units, e^2 a_0^2 for E1 | none; carried only where the code printed it |
| Total angular momentum J | dimensionless | none; stored as an exact half integer, as a numerator over two, never as a floating point number |
| Parity | even or odd | none; an enumerated value, never a sign on another quantity |
| Mixing weight | dimensionless, zero to one | a percentage printed by a code is divided by one hundred and the division is recorded as a conversion |
| Nuclear charge Z, charge state | dimensionless integers | none |

The energy conversions, and where each constant comes from:

- cm^-1 to cm^-1 is the identity, and it is still written down, because a reader
  has to be able to tell a value that was converted from one that was not.
- eV to cm^-1 multiplies by `e / (h c)`. Since the 2019 revision of the SI, `e`,
  `h` and `c` are defined exactly, so this factor is exact rather than measured:

      python -c "e=1.602176634e-19; h=6.62607015e-34; c=299792458.0; print(e/(h*c)/100)"
      8065.543937349211

  Run on 2026-08-07. The three inputs are the SI defining constants and carry no
  uncertainty.
- Hartree to cm^-1 multiplies by twice the Rydberg constant expressed in cm^-1,
  since `E_h = 2 R_inf h c`.
- Rydberg to cm^-1 multiplies by the Rydberg constant expressed in cm^-1, since
  `1 Ry = R_inf h c`.
- Atomic units of time to seconds, for transition probabilities, uses the atomic
  unit of time from the same constant set.

The Rydberg constant and the atomic unit of time are measured quantities, not
defined ones, so the record fixes where they come from rather than pasting
digits into prose that would then drift. The constant set is CODATA, one named
revision, and the revision is written into the run record of every bundle. The
numeric values live in exactly one table in the source, each carrying the CODATA
identifier it was taken from and the revision, and no other place in the tree
holds a copy. Changing the revision is a schema version bump, because it moves
every converted number in every bundle written afterwards.

Wavelengths that a code printed as air wavelengths are stored with the medium it
stated and are not silently converted. Converting air to vacuum needs a
refractive index formula, and which formula was used is a claim about the number
that has to travel with it. Where a conversion happens it happens later, in the
comparison step, it names the formula, and the result is marked as derived.

### Schema versioning, and a bundle written under an older version

The bundle schema carries its own positive integer version, counted separately
from the case schema in 0002. It is bumped by any change that makes a previously
valid bundle invalid, changes what a field means, changes which unit a field is
in, or changes the constant revision behind a conversion.

Every released bundle version stays readable indefinitely. This is stricter than
the two year window 0002 gives case files, and deliberately so: a case file is an
input somebody can rewrite, while a bundle is an archival artefact that may be
attached to a publication, cited, and reread by somebody who has no way to
regenerate it. Dropping a bundle version would break a citation, so the project
does not plan to do it, and doing it anyway would be a decision needing its own
record.

A bundle is never rewritten in place. Migrating one produces a new bundle with a
new identity, carrying the identity of the bundle it came from and the version it
was migrated from, so a reader can always reach the bytes the code actually
produced.

A reader meeting a bundle version it does not know refuses it and names both
versions. It does not read a newer bundle partially, for the same reason 0002
gives for a newer case.

## Why

The reciprocal centimetre is the unit the observational literature and the level
databases use, so an operator can hold a result next to the NIST Atomic Spectra
Database and read the difference without converting anything. Choosing
electronvolts would be friendlier to the plasma side and would force a conversion
in exactly the comparison this project exists to support.

Keeping the code's own printed value next to the converted one is not redundancy.
Conversion is the first place a rounding difference can look like a physics
difference, and having both means the question of whether the code said this or
whether the harness computed it is always answerable from the artefact alone.

Native labels are kept verbatim and never normalised on the way in. Label
translation is the exact step where a bench silently invents agreement, and doing
it inside a reader would hide it inside the least reviewed part of the system.
Whatever translation is needed happens later, in the identification step, where
it is visible and where it is allowed to fail.

The mixing weight of the leading configuration is carried because it is the only
honest handle for identification across codes. A level that is 51 per cent one
configuration and a level that is 94 per cent the same configuration are
different objects for the purpose of matching, and a model that stores only the
label cannot tell them apart.

Storing J as a half integer rather than as a float is a small rule with a large
consequence. J is a label, matching on it has to be exact, and a floating point
0.5 that arrives from a text field as 0.49999999 turns an exact label into an
approximate one at the one step where approximation is least wanted.

The constants living in one table with a named revision, rather than in prose, is
the same argument the significant digits rule in 0008 makes: a number in a
document cannot be executed, so it drifts against the number that is.

## What was rejected

Adopting one code's native format as the common one. It saves a reader and it
makes that code the reference, which contradicts the premise that none of them
is.

Storing only the converted energy. Smaller and cleaner, and it destroys the
evidence for its own correctness.

Normalising labels to a single spectroscopic notation at read time. Attractive,
and it moves the hardest judgement in the project into the place where it is
least visible.

Electronvolts as canonical. Better for the fusion and X ray audience, worse for
checking against the reference databases, and the checking is what makes the
bench trustworthy.

Converting air wavelengths to vacuum inside the reader. It would make the column
uniform, at the cost of putting an unnamed refractive index formula into the
least reviewed part of the system, which is the same mistake as normalising
labels there.

Pinning the constants by pasting their digits into this record. It reads better
and it puts a second copy of a number in a place no test can reach.

## What this costs

The schema is wider than any single code fills, most fields are optional, and
that pushes real work onto validation and onto the report, which has to render a
mostly empty row without looking broken.

Storing values twice makes the bundles larger and makes the schema look
repetitive to a reader who has not yet hit the case it exists for.

Promising to keep every bundle version readable indefinitely is a commitment the
project cannot withdraw quietly, and it will eventually mean carrying a reader
for a shape nobody writes any more.

Keeping air wavelengths unconverted means some comparisons cannot be made at all
until the operator says which formula to use, and that will look like a missing
feature rather than a refusal, every time.
