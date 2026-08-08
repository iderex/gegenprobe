# 0007. The results bundle and the rendered report are separate artefacts

Number: 0007
Title: The results bundle and the rendered report are separate artefacts
Status: accepted
Date: 2026-08-07

## What was decided

A run writes a results bundle. The bundle is the artefact. The human readable
report is rendered from the bundle by a separate command, only from the bundle,
and nothing in the renderer reads a code's raw output or reruns anything.

Raw code output is kept beside the bundle, unmodified, and is not part of the
checksummed content.

The bundle is byte stable. Two runs of the same case against the same container
manifests, both completing the same way, produce identical bytes everywhere
except in one fenced object whose fields are listed below.

### The layout

A run directory holds two things and nothing else at the top level:

    <run-dir>/
      bundle/
        manifest.json
        case.json
        run.json
        result/<participant>.json
        identification.json
        comparison.json
      raw/
        <participant>/...

`bundle/` is the checksummed content. `raw/` is not. A consumer that wants to
know what the run concluded reads `bundle/` and never needs `raw/`; a consumer
that wants to know whether a reader was wrong needs `raw/` and cannot get the
answer anywhere else.

Every file under `bundle/` is canonical JSON in exactly the form 0002 fixes for
the case file: RFC 8259, UTF-8, no byte order mark, no insignificant whitespace,
the whole document on one line, object keys sorted ascending by Unicode code
point, arrays in a defined order, and no trailing newline. There is one
canonicalisation in this project and the bundle uses it rather than a second one.

`case.json` is the canonical case, byte for byte the same bytes whose SHA-256 is
the case identity. It is not a re-serialisation and a consumer may hash it and
expect the identity to match.

`result/<participant>.json` is one file per participant, holding that
participant's levels and transitions in the common data model of 0004, with the
significance and absence markers of 0008 and 0011 already applied. `<participant>`
is the participant identifier from the case, and the file name is the only place
that identifier is encoded into a path.

`identification.json` is the output of 0005: the match groups, the unmatched
levels with their reasons, the ambiguous groups with every candidate named, the
thresholds in force, and the marks on any row an operator supplied mapping
decided.

`comparison.json` is the statistics 0006 permits, over the confident groups only,
with the tolerances in force and their `declared` or `default` marking.

`run.json` is the record of what was actually run: the participants, the manifest
each participant's image carried under 0003, the engine name and version, the
platform, the harness version and commit, the tolerances in force, the limits
that were set, whether any of them fired, and the exit status of every step. It
also holds the one fenced object described below and nothing time dependent
outside it.

### The manifest

`manifest.json` is what a third party consumer reads first, and it is written so
that reading it is enough to verify the bundle without this record in hand.

- `bundle-format`, a single positive integer, versioned by the same rule 0002
  gives the case schema: bumped by any change that can make a previously valid
  bundle invalid, change what a field means, or move the canonical bytes of a
  bundle that did not itself change. A consumer meeting a number it does not know
  refuses the bundle and names both numbers rather than reading on.
- `case-id`, the SHA-256 of `case.json` in lowercase hexadecimal, never
  truncated. It is the same string 0002 defines as the case identity.
- `participants`, the ordered list of participant identifiers, sorted ascending
  by code point, which is the order every array keyed by participant uses
  throughout the bundle.
- `written-by`, an object with the harness `version` and the `commit` it was
  built from. It names software and no person.
- `hash`, the name of the digest used for every checksum in the bundle. It is
  `sha256` today. It is a field rather than an assumption so that replacing it
  later is a bundle format bump and not an archaeology problem.
- `members`, the list of every file under `bundle/` other than `manifest.json`
  itself, each entry carrying `path` relative to `bundle/` with forward slashes,
  `bytes` as an integer, and `digest` as lowercase hexadecimal. The list is
  sorted ascending by `path`. A file present under `bundle/` and absent from
  `members` invalidates the bundle, and so does the reverse; the list is the
  authority for what the bundle contains.
- `digest`, the checksum over the members: the digest of the concatenation, in
  `members` order, of each member's `path`, a single `\n`, its `digest`, and a
  single `\n`. It is defined over the listing rather than over the file bytes so
  that a renamed file changes it.
- `variable`, the list of JSON pointers into `run.json` naming every field
  permitted to differ between two runs of the same case. The list travels in the
  artefact so that a consumer diffing two bundles does not have to have read this
  record to know what to ignore.
- `stable-digest`, computed exactly as `digest` is, over the same members, except
  that `run.json` is first rewritten with every field named in `variable`
  replaced by JSON `null` and re-canonicalised. Two runs of the same case against
  the same container manifests, both completing the same way, have equal
  `stable-digest` values. That equality is the byte stability claim, in a form a
  consumer can check with one comparison.

### The fields permitted to vary, and why each is needed

They live in one object, `run.json`'s `variable` object, and nowhere else. No
other field in any bundle file carries a clock reading, a duration, a measured
resource figure or an identifier derived from any of them. That fencing is the
whole reason a diff of two runs shows physics.

- **`/variable/run-id`.** An identifier for this run, unique within the operator's
  collection. Needed because two runs of the same case against the same manifests
  are otherwise indistinguishable by content, and an operator with a directory of
  them has to be able to say which one a report came from. It is the one field
  here that is not a time reading and it is here because it is derived from one.
- **`/variable/started`** and **`/variable/finished`**, the start and end of the
  whole run, as RFC 3339 timestamps in UTC. Needed to place the run against
  everything outside the bundle: a machine that was rebooted, a source that was
  replaced, a code whose upstream changed. Without them a bundle cannot be
  ordered against anything, including another bundle.
- **`/variable/step/<participant>/started`** and
  **`/variable/step/<participant>/finished`**, per participant. Needed to read a
  timeout. A run record saying a step was killed at its limit is unusable if the
  reader cannot see that the step ran for the full limit rather than dying at
  once, and those are the two shapes a killed step takes.
- **`/variable/step/<participant>/observed`**, an object holding the peak
  resident memory and the CPU time the step actually consumed. Needed for the
  same reason and for the case the timestamps cannot cover: a step that finished
  inside its wall clock limit while sitting against its memory limit is a run
  whose result should be read with suspicion, and only the observed figures say
  so.

Nothing else is on this list. In particular, whether a limit fired, what exit
status a step reported, and which container manifest an image carried are not
permitted to vary. Two runs differing in any of those are two different runs,
their bundles differ, and that difference is the artefact doing its job rather
than a failure of stability. The byte stability claim is about runs that
completed the same way, and it says so here so that nobody reads it as a promise
that a timeout is invisible.

## Why

Results outlive presentation. A report format will change several times, and
every change would invalidate stored results if the report were the artefact.
Rendering from a stable bundle means a two year old run can be redisplayed in
today's format without being rerun, and for cases that take hours that is the
difference between an archive and a pile of PDFs.

Byte stability is what makes the bundle diffable, and diffing two runs is the
main way anybody will use this in practice: change one thing, see what moved. A
run that scatters a timestamp through every record makes that diff useless, which
is why the time dependent fields are fenced into one object rather than removed
altogether. Removing them would cost the ability to read a timeout, which is a
real diagnostic need.

Publishing the fenced list inside the manifest, rather than only here, is what
makes the property usable by somebody who has never read this repository. A
consumer can compute the stable digest from the manifest's own instructions, and
a future field added to the fence appears to that consumer automatically.

Forbidding the renderer from touching raw output is a small rule with a large
effect. The moment a report can reach past the bundle, the bundle stops being the
complete record of the run, and nobody notices until they try to reproduce a
figure and find the number came from somewhere that was never checksummed.

Keeping the raw output is not optional. When a reader is wrong, the bundle is
wrong in the same way, and only the untouched bytes can settle it. Leaving it out
of the checksum keeps it available without making it part of the identity, since
some codes write genuinely irreproducible bytes into their own logs and a bundle
that inherited that would never be stable.

Defining the bundle digest over the member listing rather than over concatenated
file content means a rename is a change. A bundle whose files were shuffled
between participants would otherwise hash identically to a correct one, and that
is precisely the corruption that would be hardest to see by eye.

## What was rejected

A single report artefact with the data embedded. Simpler to hand around, and it
welds the results to one presentation, which is the thing this record exists to
prevent.

A database rather than files on disk. Better for querying across many runs, and
it puts a service between the operator and their own results, which is the wrong
shape for a tool meant to run on a laptop or a login node. Worth revisiting if
cross run querying becomes the main use, with evidence that it has.

Including raw output in the checksum. Stricter, and it would make bundles
unstable for reasons that have nothing to do with the physics.

Removing the time fields altogether to get stability for free. It buys a cleaner
property and it makes a fired timeout unreadable, which is a diagnostic this
project needs on exactly the runs that went wrong.

Recording the variable field list only in this record. One less field in the
manifest, and it makes the stability property unusable without the repository.

A single archive file rather than a directory. Easier to move, and it makes the
raw output either part of the identity or a second thing to carry, and it hides
the layout from anybody without the right tool.

## What this costs

Two artefacts to keep in step, and a rule about their relationship that has to be
enforced rather than assumed. Nothing in this tree refuses a renderer that reads
`raw/` today. The architecture conformance work in the quality milestone is where
that would be refused, and until it lands this is a convention.

Byte stability constrains the implementation everywhere: sorted iteration, no map
ordering in output, fixed decimal formatting, no parallelism that can reorder
results. Those constraints are cheap if adopted at the start and expensive to
retrofit, which is why this is decided before the code exists. 0008 is where they
are stated as a rule about the harness rather than about the artefact.

The manifest duplicates information a consumer could derive by walking the
directory, and a bundle whose manifest disagrees with its own contents is
invalid rather than self repairing. That is the correct handling and it means
every writer has to produce both consistently.

Keeping raw output makes a run directory much larger than its bundle, in a domain
where a single code can write hundreds of megabytes of listing. The operator has
to decide what to keep, and the bundle being separately identified is what makes
deleting the raw output a survivable choice rather than a destructive one.
