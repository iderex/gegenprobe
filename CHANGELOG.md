# Changelog

What each release changed, and what that change means for somebody holding a
bundle an earlier release wrote. The second half is the reason this file exists:
a bundle can be cited and reread long after the build that made it is gone, so a
release note that lists features and says nothing about stored artefacts leaves
the one reader who cannot ask.

How the three version numbers are chosen is `docs/versioning.md`. This file
records what a release did with them.

## The entry format

One heading per release, carrying the tool version and the date it was tagged.
Under it, these sections in this order, each naming the issue behind every line:

Added, for something the tool can now do. Changed, for something it does
differently. Fixed, for a defect repaired. Removed, for something taken away.
Any of the four is left out when it has nothing in it.

Then one more, which is never left out:

Bundles you already hold. Where the release moved `bundle-format`, this says what
a reader with an older bundle has to do, and it says it as an instruction rather
than as a version number. Where the release moved the case schema version, it
says the same for a case file somebody wrote earlier. Where it moved neither, it
says so in one line. A release that omits this section is a release that made
somebody work it out for themselves, and working it out wrongly is silent.

An entry is written for the reader outside this repository. What was refactored,
which package a thing moved into and how a check was proven are in the issue and
the pull request, and none of it belongs here.

## Unreleased

Nothing has been released. There are no tags on this repository and no releases,
so no build of this tool is in anybody's hands and no bundle written by one
exists to be held.

The work that has landed so far is scaffolding, decisions and the gate, and it is
recorded on the issue tracker and in the commit history rather than here. Entries
begin at the first release, which is #75, and the checklist that release is
worked through is `docs/release-checklist.md`.

Bundles you already hold: no bundle format has been released, so there is nothing
to hold. `bundle-format` stands at 1 and no build carrying it has been tagged.
