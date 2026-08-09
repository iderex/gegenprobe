# 0014. The fit component is Python with NumPy and SciPy

Number: 0014
Title: The fit component is Python with NumPy and SciPy
Status: accepted
Date: 2026-08-09

## What was decided

The parameter fit component is written in Python 3, and its numerical stack is
NumPy and SciPy. 0013 fixed that this question is asked from the fit's own
requirements rather than inherited from 0001, and this record is the answer to
it.

The local step is nonlinear least squares over the scaled parameters, taken from
SciPy's `least_squares`. The search over starting points is taken from SciPy's
global routines. That is the whole stack at the moment this is written. Adding to
it is an edit to the lock file described below, which is a change somebody reads
rather than a package that arrives with another one.

This decision does not reach the harness. The harness stays Go under 0001, its
module gains nothing, and a machine that runs the harness needs no interpreter.
The stack above exists only inside the fit component's image.

### What the choice had to answer

**Whether the language has a mature ecosystem for this problem.** The fit is
nonlinear least squares over a landscape with many local minima, which is why the
existing procedure is steered by a person. Both candidates considered carry
routines for it. What separated them is that the procedure this component has to
reproduce is the one the field already runs, the published fits it is measured
against under 0013 were produced by tooling from that same tradition, and the
scaling convention the fit varies is stated in the literature in terms a SciPy
residual function expresses directly. This is a claim about the field rather than
a measurement made in this tree, and it is written as one.

**Whether the operator has to install a runtime.** No. 0013 holds this component
to the container contract, so the interpreter and the stack are inside an image
built from a recipe here, and what the operator installs is the container engine
they already installed for the codes. The cost that remains is not an install: it
is image size, recipe build time, and one more toolchain this project has to keep
building. Under 0013 that cost was accepted before the language was known, and
choosing Python does not change its size in a way this record can measure.

**Whether the component can be driven through the same process contract as a
code.** Yes, and the answer is the same for anything with a command line
entry point. 0013 fixes the contract as one canonical JSON document in on a path
the runner fixes, one canonical JSON document out on a path the runner fixes,
rootless, no network, no environment beyond the allowlist, and the harness owning
the timeout and the resource limits. A Python process reads a file, writes a
file and exits with a status. The harness does not learn the language from any
of that, which is the property the contract exists to hold, and the manifest
carries the interpreter and the library versions the same way it carries a
compiler for a Fortran code.

**Whether the tests can be first tier.** The component's own suite is headless,
unelevated and offline: residual functions, parameter transforms and convergence
reporting are pure functions over committed fixtures, and nothing in them reaches
a display, a network or a path of its own choosing. What it cannot be is part of
this repository's gate tier as `docs/decisions/0009-three-test-tiers.md` defines
it, because that tier is the Go suite the single check command runs on a machine
carrying only the Go toolchain. The suite therefore runs where the interpreter
is, and the leg of the check command that calls it reports a skip naming what was
missing where the interpreter is absent, which is the shape the static analysis
leg already has in this tree. The harness side of the boundary is unaffected: it
is exercised through the fixture backed double, with no interpreter present at
all.

### How the environment is pinned, and how an operator checks they have it

Three things are pinned, in the recipe rather than in prose.

The base image is pinned by digest, which is the rule 0003 already imposes on
every recipe here.

The interpreter is pinned to an exact version in the recipe, not to a series.

Every package is pinned to an exact version and to a hash in a lock file
committed to this repository, and the install step is run with hash checking on,
so a package whose bytes are not the bytes the lock names refuses to install
rather than resolving to something newer.

A fourth thing is fixed for a reason that is not supply chain. The thread count
of the underlying linear algebra library is set to one in the recipe. A summation
whose order depends on how many threads happened to be available can move the
last digits of a residual, and two operators would then disagree about a number
this project publishes. That is stated here as the reason for the setting and
not as something measured in this tree.

What an operator does to check they have the pinned environment is read the
manifest the image carries, which 0013 already requires to hold the component
identity and version, the source checksum, the base image digest, the numerical
libraries with their versions, the platform and the recipe revision. The
component's image digest appears in the run record beside every fit result. Two
runs whose records name the same digest ran the same environment; two that do not
did not, and the record says so rather than leaving it to be inferred from
whether the numbers agree.

### The terms of what is being taken on

Read on 2026-08-09, each with the call that produced it:

    gh api repos/scipy/scipy/license --jq .license.spdx_id
    BSD-3-Clause
    gh api repos/numpy/numpy/license --jq .license.spdx_id
    NOASSERTION
    gh api repos/python/cpython/license --jq .license.spdx_id
    NOASSERTION

`NOASSERTION` is what that call answers where the terms are not in a bare licence
file the service recognises. It is not a statement that the terms are unclear and
it is not a statement that they are permissive. What each of the two says has to
be read from the project's own file before the recipe ships, and that reading is
work this record does not do. Entry 1 of the maintainer decision issue is where
this repository's own licence sits, and it is not settled either, so nothing here
asserts compatibility in either direction.

## Why

The reason to write this record before the component exists is the one 0013
gives: a means carried over from the last artefact is an assumption about this
one, and an unasked question is answered by default. The default here would have
been Go, and Go is the wrong answer for this problem for a reason that has
nothing to do with Go being a bad language. The harness's work is process
control, which is why 0001 chose it. The fit's work is numerical optimisation
over a rough landscape, and the routines, the conventions and the published
results it has to reproduce are not in the Go ecosystem.

The reason to choose the more ordinary of the two candidates is that this
component's first success criterion, fixed by 0013, is agreement with what the
existing interactive procedure produces on cases with a published result. That is
a reproduction task before it is a research task. A stack whose routines are the
ones the published work used removes a whole class of question about whether a
disagreement came from the method or from the implementation of the method, and
that class of question is the expensive one, because it is not answerable by
reading either side.

The reason the container contract does most of the work here is that it makes the
runtime question small. The argument that would normally decide against a
scientific runtime, that an operator should not need one, is answered by the
operator never installing it. What is left is a build cost and a maintenance
cost, both of which 0013 accepted when it made the fit a separate component.

The reason the thread count is pinned in the recipe rather than left to the
machine is that determinism under 0008 is a property this project asserts about
its own numbers. A component that is deterministic given its input, on a machine
whose core count nobody recorded, is deterministic in a sentence and not in fact.

## What was rejected

Julia. It is the strongest alternative and the case for it is real: it is already
in this project's plan, because JAC is a Julia package and the recipe for it under
0003 already builds a Julia environment, so choosing Julia here would reuse a
toolchain the tree carries anyway. Its licence is the cleanest of anything
considered:

    gh api repos/JuliaLang/julia/license --jq .license.spdx_id
    MIT

It was rejected on the reproduction criterion rather than on the language. The
published fits this component is first measured against were not produced with
it, its optimisation packages are younger, and a first component whose job is to
reproduce a known answer is the wrong place to spend the difference. Reuse of the
JAC toolchain is a smaller saving than it appears, because 0013 requires this
component's image to be pinned and manifested on its own terms regardless of what
another recipe happens to install.

Go with a numerical library. It would have kept the project to one language, one
toolchain and one test tier, which is worth a great deal. It was rejected because
the global optimisation and nonlinear least squares surface this problem needs is
not there in the form the field uses, so the component would spend its first
months reimplementing routines rather than reproducing published fits, and every
disagreement with a published result would have two possible causes instead of
one.

Fortran, against the existing procedure directly. It is what the procedure being
replaced is written in, and it would make the comparison exact. It was rejected
because the thing this component adds is the automatic steering, which is
programme logic rather than numerics, and because a Fortran component would be
the hardest of the candidates to hold to the input and output contract 0013
fixes without writing a second layer to do exactly that.

Deferring the choice until the component is started. This is what 0013 refuses in
advance, and it is worth naming again as the alternative it is. Deferred, the
question gets answered by whoever writes the first script.

## What this costs

A second language in a project that has argued hard for one. That is the largest
cost and it is not reduced by the container contract, because the maintenance is
paid by whoever keeps the recipe building, not by the operator.

A test tier that this repository's single check command cannot fully run. The leg
that calls the component's suite reports a skip where the interpreter is absent,
which means a green run on a machine with only the Go toolchain covered less than
the whole set and says so. That is the honest shape, and it is still a gate that
proves less than it would if the project had one language.

An interpreted stack whose exact reproducibility depends on things a version
number does not capture. The lock file, the image digest and the thread count are
the answer to that, and they are three things to keep correct rather than none.

A licence position that is not resolved by this record. Two of the three calls
above answer `NOASSERTION`, this repository has no licence of its own yet, and a
recipe cannot state what it is offered under until both halves are read. Nothing
here is blocked by that today, because the component does not exist. It blocks
the recipe.

Nothing in this record is enforced. The fit component does not exist, there is no
recipe for it, there is no lock file and no leg calls a suite that is not there.
This is what the fit milestone is required to implement, and until it does, it is
a description.
