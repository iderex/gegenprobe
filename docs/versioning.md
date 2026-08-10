# Versioning

Three things in this repository carry a version, and none of them can be derived
from the others: the tool, the case schema, and the bundle schema. One number
over all three would hide the difference between a release that adds a
subcommand and a release that cannot read a bundle its predecessor wrote, and
that difference is the one somebody holding an old bundle actually needs.

This file says how each number is chosen and what moving it means. The two schema
rules were settled in decision records and are pointed at rather than repeated,
because a rule written down twice drifts in one of the two places and nobody can
tell which.

## The tool

The tool version is a tag of the form `vMAJOR.MINOR.PATCH`. A release build
stamps it through the linker, and a build without the flag reports the commit it
came from:

```
go build -ldflags "-X github.com/iderex/gegenprobe/internal/version.stamped=v0.1.0"
```

That mechanism is `internal/version/version.go` and it is already in the tree.
What this file adds is what each of the three numbers means.

PATCH moves where a defect was repaired and nothing an operator writes or reads
changed. Neither schema number moves in such a release.

MINOR moves where the tool can do something it could not before: a subcommand, a
participant, a reader, a flag, a field in the report. Either schema number may
move in the same release, and the changelog entry is where that is said.

MAJOR moves where something an operator was relying on is removed or changed
under them. While the tool is at version zero this number does not move, and what
that costs is the next section rather than an implication.

The tool version is not a statement about either schema. A build can gain a
subcommand without touching a bundle, and can bump `bundle-format` in a patch
release if a defect turns out to be a wrong field meaning. Reading one number off
the other is what the three separate numbers exist to prevent.

## Version zero, and what may break

The tool is at version zero, and nothing has been released:

```
gh release list -R iderex/gegenprobe --limit 5 ; echo "exit=$?"
exit=0
git tag | wc -l
0
```

No output from the first command, and no tags. So every statement below is about
a promise being made rather than one being kept so far.

While the major number is zero, the tool promises nothing about itself between
releases. Named plainly, because the point of stating it is that somebody can act
on it: a subcommand may be renamed or removed; a flag may change what it means;
anything the tool prints, including the rendered report, may change in any way; an
exit status may change; and the layout of a bundle directory may change under a
`bundle-format` bump. A script built on any of those is a script that breaks, and
the version number will not warn it first.

Two promises are kept at version zero anyway, because they are about artefacts
rather than about the tool.

A released bundle format stays readable indefinitely, which is stricter than
anything the tool promises about itself, and the reason is in
`docs/decisions/0004-the-common-data-model.md`: a bundle can be attached to a
publication and reread by somebody with no way to regenerate it.

A released case schema version stays readable for at least twenty four months
after the release that replaced it, which is `docs/decisions/0002-the-case-file.md`.

Neither promise can be honoured by silence, so a build meeting a version it does
not know refuses the artefact and names both numbers rather than reading on.

## The case schema

The case file carries `version`, a single positive integer, which is the version
of the case schema and not of anything else. What bumps it, what a loader does
with a version it does not know, and how long a version stays readable are in
`docs/decisions/0002-the-case-file.md`.

There is no case schema in the tree yet. #29 is what writes the first one, so
nothing carries this number today and the rule above is a rule waiting for its
first artefact.

## The bundle schema

A bundle's manifest carries `bundle-format`, a single positive integer, counted
separately from the case schema. What bumps it is in
`docs/decisions/0004-the-common-data-model.md`, and the field is described among
the rest of the manifest in `docs/decisions/0007-bundle-and-report.md`.

It is at 1, held in `internal/model/model.go`, and the refusal a reader owes a
version it does not know is `internal/model/read.go`. The refusal names both
numbers and does not read a newer bundle partially:

```
go test ./internal/model -run 'TestABundleUnderAnOlderFormatIsRefusedNamingBothVersions|TestABundleUnderANewerFormatIsRefusedAndNotReadPartially|TestAManifestWithNoFormatFieldIsRefused' -count=1 -v
=== RUN   TestABundleUnderAnOlderFormatIsRefusedNamingBothVersions
--- PASS: TestABundleUnderAnOlderFormatIsRefusedNamingBothVersions (0.01s)
=== RUN   TestABundleUnderANewerFormatIsRefusedAndNotReadPartially
--- PASS: TestABundleUnderANewerFormatIsRefusedAndNotReadPartially (0.00s)
=== RUN   TestAManifestWithNoFormatFieldIsRefused
--- PASS: TestAManifestWithNoFormatFieldIsRefused (0.00s)
PASS
ok  	github.com/iderex/gegenprobe/internal/model	1.822s
```

Those tests live beside the type they judge rather than beside this document, and
this file points at them for the reason the rest of it points at records: two
places asserting one property can disagree, and then the property is whatever the
weaker of them says.

## What reads this file

Nothing does. No leg of `go run ./gate` reads this document, so the scheme above
is held in step with the tree by whoever writes a release rather than by a check.
The one part of it that is refused by a machine is the bundle format mismatch,
and that is a check on the code and not on the sentences here.

`CHANGELOG.md` is where a release says which of the three numbers it moved, and
`docs/release-checklist.md` is the order the steps are worked through.
