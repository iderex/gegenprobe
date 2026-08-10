# The fit: what is varied, against what, and what makes a fit good

The parameter fit is the second component of this project. Record 0013 makes it
a separate component and holds it to the same process contract a code is held
to, and record 0014 fixes what it is written in. Neither says what is being
optimised. This document does, and it is written before any optimiser exists,
because the reason the existing procedure is run interactively is that a person
steers it, and an automatic procedure that has not been told where the person
was steering will arrive somewhere else and report a smaller number.

Everything below is a statement of the problem. Nothing in this tree implements
any of it and nothing refuses a violation of it.

## What is varied

### The parameters of the Hamiltonian

The fit varies the parameters of a Slater-Condon Hamiltonian over a declared set
of configurations. The parameter classes are the ones the field uses, and they
are named here so the rest of this document can refer to them without
re-deriving them:

| Class | What it is | Sign |
| --- | --- | --- |
| `E_av` | the average energy of one configuration | either |
| `F^k(nl,n'l')` | direct Coulomb integrals, `k` even for equivalent electrons | positive |
| `G^k(nl,n'l')` | exchange Coulomb integrals | positive |
| `zeta_nl` | the spin-orbit parameter of one subshell | positive |
| `R^k(...)` | configuration interaction integrals between two configurations | either |
| `alpha`, `beta`, `T` | effective parameters standing for many-body effects inside a shell of equivalent electrons | either |

The signs in the third column follow from the definitions of the integrals
rather than from anything measured here, and they are the defaults the
constraint section below applies. A case may override any of them, and where it
does, the override is written into the output beside the result.

This document does not fix which configurations enter. That is the case's
business, it is the largest single lever on how long the fit takes and on how
well it does, and a component that chose it would be choosing the answer.

### The three states a parameter can be in

Every parameter in the declared set is in exactly one of three states, and the
state is declared in the input rather than discovered by the component.

Free. The parameter has its own varied quantity and the optimiser moves it
alone.

Linked. The parameter belongs to a named group. One varied quantity moves the
whole group, and the ratios between the members of a group are held at the ratio
their starting values had. Linking is what makes a fit possible at all where
there are more parameters than the observed levels can determine, and it is a
declaration about which parameters the data cannot separate, so it belongs to
whoever wrote the case.

Fixed. The parameter keeps its starting value and the optimiser does not move
it. A fixed parameter is still reported, with its value and with the fact that
it was fixed, because a reader comparing two fits needs to see that a parameter
one of them determined was held in the other.

There is no fourth state. In particular there is no state in which the component
decides for itself that a parameter is undetermined and freezes it mid-run: a
parameter the data cannot determine comes out of the fit with an uncertainty
larger than itself, which is a result somebody reads, rather than being removed
from the fit by the thing being measured.

### What is varied is a factor, not a parameter

The varied quantity for a free parameter, and for a linked group, is a
dimensionless factor multiplying the starting value, not the parameter itself.

    P_i(x) = x_i * P_i(start)

This is the convention the field states its results in, and it is the one that
makes a bound meaningful: a bound on a factor says how far from the ab initio
value the fit is allowed to travel, and it means the same thing for two
parameters that differ by three orders of magnitude. `E_av` is the exception,
because it is an offset rather than a scale and a factor on it is a factor on
an arbitrary zero: `E_av` is varied as an additive shift in the canonical unit,
one per configuration.

A starting value of zero cannot carry a factor. Where a parameter starts at
zero, the case declares it fixed or gives it a non-zero starting value; the
component refuses the input rather than dividing by it.

### Where the starting values come from

The starting values are the ab initio parameters of an atomic structure
calculation over the same configuration set, supplied in the input document. The
component does not compute them, does not call a code to compute them, and has
no default for them. That keeps the fit reproducible from its input alone, which
is what 0013's process contract requires, and it keeps the choice of which code
produced the starting values visible in the record rather than buried in the
component.

## What it is varied against

### Which observed levels enter

The observed level energies are the ones record 0013 puts in the model, each
attached to a calculated level, each carrying its own source, its own
uncertainty and its own `association`.

A level enters the objective when the case says it does. The case may exclude a
level, and where it does, the exclusion carries a reason that is written into
the output. Excluding a level is a legitimate act and a silent one is not: the
number of levels in the objective and the number attached but excluded are both
reported, so a fit that reached a low deviation by dropping the levels it could
not reproduce says so in its own output.

Where a level carries more than one observed entry, which 0013 admits on purpose,
the case names which entry enters. The component does not average two
compilations and does not choose between them.

### How a calculated level is attached to an observed one

The attachment is declared, never computed. Record 0013 fixes the vocabulary:
`by-operator` where the case named the pairing, `by-source-identifier` where the
source keys on the same level identity the case used, and no third value. There
is no attachment by energy proximity, because a residual computed against the
nearest observation is a function of the attachment rather than of the physics,
and it gets smaller as the calculation gets worse.

This is where the fit differs from record 0005, and the difference is worth
stating because 0005 is the identification decision this document is required to
be consistent with. 0005 governs the matching of one code's levels to another
code's levels, and it runs a matcher: blocks keyed by `J` and parity, canonical
leading configurations, a one-to-one assignment inside a block, with unmatched
and ambiguous as outcomes. None of that runs here. The calculated-to-observed
pairing is an input to the fit, not a step of it.

The two are consistent rather than merely different, and in three ways that
matter. Neither reads a code's printed spectroscopic label. Neither attaches
anything by energy proximity. Both treat a pairing nobody can make as an outcome
to report rather than a gap to fill: 0005 reports it as unmatched, and here an
observed level with no declared attachment simply does not enter the objective
and is counted in the output as attached to nothing.

One thing does have to be recomputed at every step, and it is not the
attachment. The attachment names a calculated level; a diagonalisation produces
eigenvalues, and which eigenvalue belongs to that level changes when two levels
cross during the iteration. The rule is:

- Inside a symmetry block keyed by `J` and parity, the eigenvectors of the
  current iterate are matched to the eigenvectors of the starting
  diagonalisation by a one-to-one assignment maximising the total squared
  overlap. Eigenvector composition is a computed property of the current
  Hamiltonian; a printed label is not, and this rule reads the first and never
  the second.
- Energy order is not used, for the same reason 0005 refuses it: the ordering is
  a thing the fit is allowed to change, and using it to decide what is being
  compared would build the answer out of itself.
- Every step at which the assignment differs from the previous step's is counted,
  and the count is reported with the result. A fit that reordered its levels
  fourteen times on the way to its minimum is a different object from one that
  never did, and a reader cannot see that in the deviation.
- Where the best assignment and the next best are separated by less than the
  declared overlap margin, the step is recorded as ambiguous, with the block and
  the competing levels named. The default margin is a declared parameter of the
  input with no default this document fixes, because nothing in this tree has
  measured what a sensible one is.

### How they are weighted

The residual for one attached level is

    r_j = E_j(calculated, relative to the ground level) - E_j(observed)

in the canonical unit of record 0004, which is the reciprocal centimetre, and
which is also the unit the observational literature quotes. Both sides are
energies relative to the ground level of the ion. An absolute energy does not
enter, because the observed values are level positions and the calculated
`E_av` carries a zero the observation has no counterpart for.

The weight is declared per level, and where the case declares none the default
depends on what the observation carries:

- `uncertainty.kind` of `standard`, and a `value`: the weight is the inverse
  square of that value.
- `uncertainty.kind` of `expanded`, with a `coverage-factor`: the weight is the
  inverse square of the value divided by the coverage factor.
- `uncertainty.kind` of `stated-interval`: the case declares the weight. The
  component does not convert an interval into a standard uncertainty, because
  doing so is an assumption about a distribution the source did not state.
- `uncertainty.kind` of `unstated`: the case declares the weight, and where it
  declares none the level enters with unit weight and the output says how many
  levels did so. A unit weight beside inverse-square weights is a statement that
  the level counts as much as an observation with an uncertainty of one
  reciprocal centimetre, which is usually not what anybody meant, and it is
  reported rather than hidden.

### A level whose experimental assignment is uncertain

An observation whose assignment to a state is doubted in its own source is a real
and common case, and it is the case in which a fit most easily talks itself into
a wrong minimum.

The rule is that this is declared and never inferred. The case marks such a level
in one of two ways, and the output reports the counts under both:

- Excluded, with the reason. It contributes nothing to the objective and it is
  still reported, with the deviation the fit produced for it, which is the number
  a reader wants when deciding whether the doubt was justified.
- Included with a declared weight lower than its uncertainty alone would give.
  The declared weight and the weight the uncertainty would have given are both
  reported, so the size of the thumb on the scale is visible.

There is no third way, and in particular the component never decides that a level
is a poor fit and therefore probably misassigned. That inference runs the wrong
way: it uses the calculation to correct the observation, and once it is available
the fit can improve its own deviation by disbelieving the data.

## The objective

The objective is the weighted sum of squared residuals over the levels that
entered:

    S(x) = sum_j w_j * r_j(x)^2

The local step is a nonlinear least squares minimisation of that sum over the
varied factors, subject to the constraints below. Record 0014 fixes what
computes it.

The objective contains nothing else. It carries no penalty term drawing the
parameters towards their ab initio values, no term rewarding a smaller parameter
set, and no term over transition rates. Each of those is defensible and each
changes what the number at the end means: a deviation from a penalised objective
is not comparable with a published deviation from an unpenalised one, and
comparability with published fits is the first success criterion record 0013
sets. A case that wants a parameter held near its starting value has a
constraint for it, which is a bound the reader can see, rather than a term
trading against the data by an amount nobody quoted.

## The constraints

### Bounds

Every varied factor carries a closed interval, declared per parameter or per
class. A parameter whose fitted value sits on its bound at the end is reported
as such, because a bound that is active is a constraint deciding the answer
rather than the data deciding it.

This document fixes no default interval. The scaling convention the field uses
puts the fitted Coulomb parameters below their ab initio values, and how far
below is a property of the ion and of the configuration set rather than a
constant, so a default written here would be a number invented in this tree and
then quoted back as if it had been measured.

### Signs

A parameter class with a fixed sign in the table above keeps it. The constraint
is on the parameter and not on the factor, so a negative starting value with a
positive factor is refused at validation rather than at the first iteration.

### What the optimiser is not allowed to do

The following are refusals, not preferences. Each of them lowers the objective
and none of them is a better fit.

- Change which levels are in the objective. The set is fixed by the input before
  the first iteration and does not move.
- Change the attachment between calculated and observed levels.
- Free a parameter the case fixed, unfix a link, or change the ratio inside a
  group.
- Change the configuration set, including by dropping a configuration that
  contributes no observed level.
- Move a parameter outside its bounds or across its declared sign.
- Take a starting point, a seed or a tolerance from anywhere but the input
  document. Record 0013 requires the seed of a stochastic step to come from the
  input and to be recorded in the output, and a run whose seed came from the
  clock is a run nobody can repeat.

## What makes a fit good

### The number the field uses

The root mean square deviation of the fitted eigenvalues from the observed
levels, in the canonical unit, over the levels that entered the objective. It is
reported unweighted and weighted, both, because the two answer different
questions and quoting one as the other is the ordinary way this number is
misread.

It is not sufficient on its own. A fit with a low deviation and a parameter set
outside the range the ion admits is a worse answer than a slightly worse fit that
stayed inside it, and nothing about the deviation says which one happened.

### What is reported beside it

All of the following, in the output document, as numbers rather than as prose a
reader has to count:

- The number of observed levels attached, the number that entered the objective,
  and the number excluded with the reason for each.
- The number of varied quantities, free and linked counted separately, the number
  of fixed parameters, and the difference between the number of levels and the
  number of varied quantities.
- The deviation broken down per symmetry block and per configuration, beside the
  total. A total that hides one badly reproduced configuration is the failure
  this breakdown exists to show.
- The largest residuals, with the levels they belong to, so the worst case is
  visible without recomputing anything.
- Each varied quantity's final value with its standard deviation from the
  covariance matrix at the minimum, and the count of those whose value is not
  separated from zero by their own standard deviation.
- Each parameter that finished on a bound.
- The count of eigenvector reassignments during the iteration, and every step
  recorded as ambiguous under the overlap rule.
- The convergence status, and the seed where a stochastic step was used.
- The starting values, and their source, since a deviation without them is not
  reproducible.

Where the fit did not converge, the affected cells are absent with the reason
`not-converged`, which record 0013 fixes as the ordinary case here rather than
as an error.

### The acceptance criterion

Record 0013 fixes it and this document does not soften it. The first criterion is
agreement with what the existing interactive procedure produced on a case with a
published result: the parameters this component finds agree with the published
ones inside the published uncertainties, and the deviation is no worse than the
published one by more than the margin the reference case declares.

A lower deviation than the published fit is not a pass. It is a result that has
to be argued for, and the argument has to say which of the two fits has the
parameter set the ion admits, because the cheapest route to a lower deviation is
a parameter set nobody would defend.

## A published fit, worked through

The reference below is one published fit read in the vocabulary this document
sets up. It is here so that the statement above is anchored to something real
rather than to a description of a procedure.

Alexander Kramida, "Assessing Uncertainties of Theoretical Atomic Transition
Probabilities with Monte Carlo Random Trials", Atoms 2 (2014) 86 to 122,
[doi:10.3390/atoms2020086](https://doi.org/10.3390/atoms2020086), read on
2026-08-10 at
[the open access copy](https://pmc.ncbi.nlm.nih.gov/articles/PMC4889025/). The
quotations below are that paper's own sentences.

The ion is titanium-like iron, Fe V. The configuration set is `3d^4`,
`3d^3(4s + 5s + 4d + 5d)` and `3d^2(4s^2 + 4s4d + 4d^2)`.

What was varied, in that paper's words: "86 Slater parameters (average energy
E_av, spin-orbit parameters ζ3d and ζ4d, direct and exchange Coulomb interaction
parameters F2,4(nd,n′d) and G0,2,4(nl,n′l′), respectively, and effective
parameters α3d, β3d, and T3d; those are coefficients of Casimir operators
representing many-body effects in shells with equivalent electrons, see Cowan's
book [4], section 16-7), and 61 CI parameters". Every class in the table at the
top of this document appears there, which is the point of quoting it.

The three states appear as well, and the paper states them as counts: "there
were 15 such groups, which included 121 parameters, nine parameters were allowed
to vary independently, and seven parameters were fixed in the LSF". The linking
rule is the one this document calls a group with fixed internal ratios, and the
paper's own table says where the ratios came from: "Parameters in each numbered
group were linked together with their ratio fixed at the Hartree-Fock level".
That the ratios come from the ab initio calculation, and not from the fit, is
exactly the reading this document takes.

Two counts in that paper are quoted here as printed and are not reconciled. The
parameter classes above add to 147, and the grouping sentence accounts for 137.
Which parameters the second sentence ranges over is decidable only from the
paper's own table, which is not read here, so no arithmetic is asserted over the
two.

The deviation, and the reason it is quoted twice: "The standard deviation of the
eigenvalues produced by the LSF from experimental energies is 117 cm−1 for all
220 experimentally known even levels and 41 cm−1 for the 34 levels of 3d4". One
fit, two numbers, differing by a factor near three depending on which levels are
counted. That is the breakdown this document requires beside the total, present
in a published result because the author found it worth stating, and it is the
clearest available argument that a single number is not enough.

What this example does not supply is a case in the form this project reads. It is
a published fit read in this document's vocabulary, and turning it into a
reference case with an acceptance margin is the work of the reference fits issue
rather than of this document.

## What this document deliberately does not fix

The configuration set for any case, the bounds on any factor, the overlap margin,
and the weights. All four are properties of the ion and of the observations
available for it, and a default written here would be a number this tree invented
and would afterwards be quoted as if somebody had measured it.

Which optimiser runs, and how the search over starting points is arranged. That
is the optimiser issue, and this document is the statement of the problem it has
to solve.

The form of the input and output documents. Record 0013 fixes that they are one
canonical JSON document each, on paths the runner fixes, and the field names are
settled where that shape is built rather than here.

Nothing in this document is enforced. The fit component does not exist, no input
document has a schema, and no check in this tree refuses a fit that ignores any
of it. This is what the fit milestone is required to implement, and until it
does, it is a description.
