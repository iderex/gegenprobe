# 0003. This repository ships container recipes, never images and never the codes

Number: 0003
Title: This repository ships container recipes, never images and never the codes
Status: accepted
Date: 2026-08-07

## What was decided

This repository ships container build recipes. It does not ship container images,
and it does not vendor the source of any of the codes.

A recipe takes the code's source from a location the operator supplies, records
the SHA-256 of exactly what it was given, and bakes a manifest into the resulting
image. Every run reads that manifest out of the image and copies it verbatim into
the run record, so a bundle carries the provenance of the binary that produced
its numbers rather than a claim about it.

Containers run rootless wherever the engine supports it. The harness never asks
for elevation, never invokes a tool that raises a consent prompt, and a recipe
that needs one is a defect in the recipe.

### The manifest, field by field

The manifest is the answer to "what exactly produced this number". Each field is
here because a comparison loses a specific meaning without it, and that loss is
named beside the field rather than left to be inferred.

- **`code`.** Which participant this image is. Without it the bundle has numbers
  with no owner, and a map that cannot say which code produced a column is not a
  comparison of codes.
- **`code-version`.** The version, release or revision the source declared for
  itself. Without it a disagreement between two runs of the same participant
  cannot be told from a disagreement between two versions of it, which is the
  first question anybody asks when a map changes.
- **`source-sha256`.** The checksum of the source as handed to the build, before
  anything was applied to it. This is the field that makes the operator's copy
  and somebody else's copy comparable at all. Without it "we both ran
  AUTOSTRUCTURE" is an assertion about a name, and the codes in this field
  circulate as ports and local branches under one name.
- **`patches`.** The checksum and the ordered list of any patch the recipe
  applied on top of that source, empty where none was applied. Without it the
  source checksum describes something the compiler never saw, which is worse than
  having no checksum, because it looks like provenance.
- **`base-image-digest`.** The digest, never the tag, of the image the build
  started from. A tag moves. Without the digest the libc, the shell and the
  system numerical libraries are unpinned, and two builds of identical source can
  differ for reasons no field in the bundle records.
- **`compiler`** and **`compiler-version`.** Which Fortran or C compiler, at which
  exact version. Without these a difference between two columns can be a
  difference between two compilers, and this project's whole output is a claim
  about where methods differ. A method difference and a compiler difference
  reported as the same thing is the failure this bench exists to prevent.
- **`compile-flags`** and **`link-flags`.** The complete flag sets as passed,
  including the optimisation level, the floating point model, and any default
  integer or real width. These codes are old enough that width flags change
  results rather than performance, and an optimisation level that permits
  reassociation changes the last digits of every energy. Without the flags the
  significance rules in 0008 are being applied to a number whose precision nobody
  can account for.
- **`numerical-libraries`.** Each linked numerical library with its name, version
  and, where the base image provides it, its package digest. A diagonalisation is
  the numerical heart of every one of these codes. Without this field a
  disagreement traceable to two different LAPACK implementations is reported as a
  disagreement between physical methods.
- **`platform`.** The operating system family and CPU architecture the image
  targets, for example `linux/amd64`. Without it a result that depends on
  instruction set or floating point unit behaviour cannot be reproduced, and the
  reader has no way to know that a rerun on other hardware is a different
  experiment.
- **`recipe-revision`.** The commit in this repository the recipe was built from.
  Without it the recipe itself is the one uncontrolled input in a chain that
  controls everything else, and a change to a build step becomes invisible.

The manifest carries no build timestamp and no builder identity. A timestamp
names no property of the binary, it would make two otherwise identical builds
differ, and it is exactly the kind of field that turns a diff of two runs into
noise. Builder identity is a person, and 0012 keeps people out of the artefacts.

### What reopens this

The technical half of this decision, which is that a recipe with a recorded
checksum is what proves a binary corresponds to a source, stands on its own and
does not depend on anyone's answer. The legal half does.

Entry 2 of the maintainer decisions issue asks whether this project may ever
distribute container images containing third-party code. If it is answered the
other way, this record does not quietly bend. It is superseded by a new record,
in both directions, and the new record has to carry all of the following, because
each one is a thing this decision currently does not have to say.

- Which participants images may be published for, one at a time and by name, with
  the terms each one was cleared against and the date they were read. RMATRX1
  cannot be on that list under the CPC licence, which refuses redistribution in
  original or modified form without the author's written permission
  (https://www.elsevier.com/about/policies-and-standards/open-access-licenses/elsevier-user/cpc,
  read on 2026-08-06).
- What a published image is offered under, which cannot be answered before entry
  1 of that issue fixes this repository's own licence.
- Where the images are published, who holds the credentials, and what happens to
  them when a participant's terms change after publication.
- Whether the recipe route stays available beside the image route. Removing it
  would remove the operator's ability to check the claim themselves, which is the
  standard this project asks the rest of the field to meet, so the successor
  record has to say plainly that it is doing that if it does.
- What the manifest means in an image somebody else built. The fields above
  describe a build the operator performed. In a published image they describe a
  build the operator has to take on trust, and the successor record has to say
  which of the two the bundle is recording.

Nothing here is blocked on that answer. The recipes are built either way.

## Why

The codes this harness compares do not share a licence and two of them state
none at all. The AUTOSTRUCTURE distribution page carries no licence, no
permission statement and no conditions of use
(https://amdpp.phys.strath.ac.uk/autos/). RMATRX1 carries the CPC non-profit use
licence, which entitles one licensee and their research group to a copy for
academic or non-profit use and permits no commercial use, no re-licensing and no
redistribution in original or modified form without the author's written
permission, at the address above, read on 2026-08-06. That one is not an absence
of terms, it is a refusal in as many words, and it settles the question for that
participant without anybody having to weigh anything. Cowan's chain circulates in
ports with no terms travelling alongside. Publishing images built from those
would put a question this project cannot answer into the critical path of every
user, under one person's name.

The reproducibility argument points the same way and would stand even if every
licence were permissive. An image is a claim that a binary corresponds to a
source, and only the person who built it can check that claim. A recipe plus a
recorded checksum lets the operator make the claim themselves, on their own
machine. That is precisely the standard this project asks the rest of the field
to meet, and a bench that asked for it while shipping unverifiable binaries would
be arguing against itself.

The manifest is not decoration and it is not documentation. A comparison between
two codes built with different compilers at different optimisation levels is a
comparison of four things at once, and a map that cannot say which four is not a
measurement. Baking the manifest into the image rather than writing it down
beside the image is what makes it survive being copied, moved and rerun a year
later.

Rootless is a requirement rather than a preference, because a bench that needs
administrative rights cannot run on the machines where this work actually
happens, and those include managed institutional desktops and shared cluster
login nodes. It is the same rule the gate tier is held to in 0009, arriving from
the other direction.

## What was rejected

Shipping images for everything. The fastest onboarding available by a wide
margin, and it is the largest single barrier this project has. It is not rejected
here on the merits, because the merits are not this plan's to weigh. It is entry
2 of the maintainer decisions issue, and the paragraph above says what a yes
would have to produce.

Shipping images only for the permissively licensed codes. Defensible, and it
produces a bench where half the participants install in one command and half take
an afternoon, which is a confusing thing to hand somebody who is trying to decide
whether the tool is worth their time.

Vendoring the sources into this tree. The same legal problem, plus this
repository would carry several megabytes of Fortran it does not maintain, cannot
patch and cannot answer questions about. It would also make every clone of this
repository a redistribution.

No containers at all, with the operator building each code on the host. This is
the status quo in the field, and it is exactly what makes results hard to repeat.
It also puts the build environment outside anything the run record can capture.

Recording the manifest beside the image rather than inside it. Simpler to
generate, and it separates the provenance from the thing it describes at the
first `docker save` or registry push.

## What this costs

The first run is slow and manual. The operator has to obtain each code
themselves, from several places, under several sets of terms, and some of those
require an institutional affiliation. This is the largest barrier the project
has, and it is accepted deliberately rather than by omission.

The project takes on a support burden for build failures it did not cause, in
codes it does not maintain, on compilers it did not choose. Every new compiler
release is a chance for a recipe to break in a way that has nothing to do with
this project's own work.

At least one participant, HFR+CPOL, is not publicly distributed at all so far as
this planning found, so a recipe for it may never be usable by anybody outside
the group that holds it. That is a gap and it is recorded as one, in the register
the running milestone keeps, rather than left as a silence in a list of supported
codes.

Nothing in this tree refuses an image or a vendored source today. The repository
invariant lint planned in the quality milestone is where a rule of that shape
would live, and until it exists this record is a description of what the
maintainers do rather than something a machine prevents.
