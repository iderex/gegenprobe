# 0010. JAC runs under the same contract as the Fortran codes

Number: 0010
Title: JAC runs under the same contract as the Fortran codes
Status: accepted
Date: 2026-08-07

## What was decided

JAC participates through exactly the same container contract as the Fortran
codes. The recipe builds a Julia image, the harness generates a script from the
canonical case, the container runs it, and a reader turns its output into the
common model.

No Julia is required to build, test or run the harness. The Julia runtime lives
inside one container image and nowhere else.

JAC gets no special status in the comparison. It is not the reference, it is not
excluded from disagreement statistics, and the report does not describe it as the
modern one.

### The licence position, as measured

The GitHub licence endpoint reports no assertion for JAC:

    gh api repos/OpenJAC/JAC.jl/license --jq .license.spdx_id
    NOASSERTION

Run on 2026-08-07. That is a statement about the shape of the repository and not
about the terms. The endpoint classifies a bare `LICENSE` file; JAC's licence
text sits inside a Markdown page, at
https://github.com/OpenJAC/JAC.jl/blob/master/LICENSE.md, and the text itself is
the MIT licence, with the user guide under CC BY 4.0. The distinction is worth
the paragraph, because reading `NOASSERTION` as "no licence" would be reading the
nearest artefact instead of the thing itself, and it would put JAC in the same
category as the codes that genuinely state nothing.

What this project asserts, therefore, is the endpoint output above, which it ran,
and a reading of the linked page. It does not assert that the terms have been
reviewed by anybody qualified to review terms.

What this repository's own work is offered under is not settled and is not this
record's to settle. It is entry 1 of the maintainer decisions in #1, and the
container recipes cannot state what they are offered under until it is answered.

### Pinning the Julia version

Three things are pinned, in the recipe, in the tree:

- The Julia base image, by digest rather than by tag, so that the same recipe
  builds the same runtime a year later.
- JAC itself, by release tag together with the commit sha that tag pointed at
  when the pin was made. The tag is for a reader, the sha is what is checked.
- The Julia package set, by a `Manifest.toml` committed beside the recipe, so the
  transitive dependency graph is fixed and not resolved at build time.

Precompilation happens during the image build and is baked into the image. It
never happens at run time, so no case pays for it and no case's wall clock
depends on whether an earlier case warmed something.

None of the three is ever written as a range. A range turns a reproducibility
property into a coincidence about when the image was built.

### When the pin stops matching what JAC supports

The failure lands at image build time, during precompilation, which is the point
of baking precompilation into the build. A JAC release that needs a newer Julia
than the pinned base image will not precompile, the recipe build fails, and no
case runs against a mismatched pair.

The response is to move the pins together and to prove the move, not to loosen
them. Moving them means: new base image digest, new JAC tag and sha, regenerated
manifest, and a rerun of the acceptance case with the results compared against
the previous pin. A move that changes numbers is a finding about JAC or about the
harness and is handled as one; it is not something to absorb quietly into a pin
bump.

Where the pins cannot be moved to any working pair, JAC's entry goes into the
register of participants that cannot be fully supported, #42, with what was tried
and what failed. That register exists so an unreachable participant is a recorded
state rather than a silence, and JAC is not exempt from it for being the newest
code in the bench.

## Why

Including it is the point. This project's plan is explicit that the harness does
not compete with JAC, it includes it, and a bench of four Fortran codes from
between 1974 and 2018 would be a study of one tradition rather than a comparison
of methods.

Holding it to the same contract is what keeps the cost contained. Julia's startup
time and package precompilation are real, and the natural temptation is to embed
a persistent Julia process in the harness to avoid paying them repeatedly. That
would put a second runtime inside the tool for one participant's benefit and
would breach the neutrality argument that decided the harness language.

Giving it no special status matters because it is the participant most likely to
be treated as the reference, on the grounds of being newest. Newest is not an
argument about accuracy, and this project has nothing to say about which code is
right.

Pinning by digest and sha rather than by tag is the same argument as the case
identity in 0002. A tag is a name somebody can move; a digest is the bytes. A
bench whose results cannot be tied to the exact runtime that produced them has
lost the property it exists to provide.

## What was rejected

Writing the harness in Julia so JAC integrates natively. Rejected in the language
decision, for neutrality and for the installation burden it would put on every
operator.

Calling JAC in process through an embedded runtime. Faster per case, and it makes
the harness carry a runtime for one of the codes it compares.

Leaving JAC out of the first release. Would ship sooner, and it would ship the
thing the plan explicitly said not to build.

Precompiling at first run instead of at image build. It makes the image smaller
and it puts a several minute variable cost on somebody's first case, at the exact
moment they are deciding whether this tool works.

Pinning Julia by a version range that JAC declares compatible. It would survive
more JAC releases without a change here, and it would mean two operators running
the same recipe on different days get different runtimes, which is the property
this bench cannot give up.

## What this costs

Julia's first run in a fresh container is slow, and precompilation has to be
handled in the recipe rather than at run time or every case pays for it.

JAC is under active development while the Fortran codes are largely static, so
its reader will need maintenance at a different rhythm from the others, and a
version pin in the recipe is doing more work here than elsewhere.

Pinning three things together means every JAC bump is a small piece of work with
an acceptance run attached, rather than a one line change, and the pin will
therefore lag JAC's releases. Lagging visibly is the intended trade against
tracking invisibly.

The licence position rests on a reading of a Markdown page rather than on a
machine readable assertion, so the check quoted above will keep reporting
`NOASSERTION` and will keep needing this paragraph next to it.
