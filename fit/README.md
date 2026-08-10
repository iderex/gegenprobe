# The fit component

This directory is the parameter fit, the second component of this project. It
holds no code today and nothing in it runs.

What is settled about it is settled in three places, and this file is the way
into them.

`docs/decisions/0013-the-fit-is-a-separate-component.md` makes the fit a
separate component, holds it to the same process contract a code is held to, and
fixes the schema obligation for observed level energies that the harness carries
on its behalf.

`docs/decisions/0014-the-fit-component-is-python-with-numpy-and-scipy.md` is the
means check for it, and fixes the language, the numerical stack and how the
environment is pinned.

[The fit problem](../docs/fit-problem.md) is what is being optimised: the
parameters that are varied, the observations they are varied against, the
constraints, and what makes a fit good. It is the statement of the problem, and
it is written before any optimiser so that an automatic procedure can be checked
against what the interactive one was steering towards rather than against a
smaller number.

The fit does not ship in the first release. That is 0013, and the plan issue on
the tracker says the same thing in the list of what the first release
deliberately leaves out.
