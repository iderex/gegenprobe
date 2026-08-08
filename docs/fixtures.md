# Fixtures

A fixture is recorded bytes this repository's tests are run against. It is
stored encoded, with its provenance in the same file, under a `testdata`
directory. `go run ./gate` refuses one stored any other way.

## Why not just commit the file

The readers here are tested against output from fixed format Fortran programs.
Column positions carry meaning, a trailing space can be part of a field, and
some of those files arrive with carriage returns. Every one of those bytes is
something a checkout is free to rewrite.

That is not a hypothetical about somebody else's machine. In this clone:

```
git config core.autocrlf
true
```

With that set, git stores a text file with newlines and hands the working tree
carriage returns, so a fixture collected on Linux and committed raw is a
different file by the time a test on Windows opens it. Nothing goes red. The
reader parses the damaged bytes, gets plausible numbers out of them, and the
fixture goes on testing something it was not collected to test.

A `.gitattributes` rule can turn that off, and this convention deliberately does
not rely on one. An attributes file protects the paths it names in the clones
that have it; a contributor's editor, a copy through a text field, an archive
download and a tool that rewrites on save are all outside it. Encoding the bytes
removes the question instead of answering it.

## The convention

A fixture is a file named `*.fixture` under a directory named `testdata`. It
holds a provenance note, a blank line, and the bytes in base64:

```
Code: GRASP2018
Version: 2018.1
Case: hydrogen-like-1s2p
Kept: the first two lines of the level table

ICAxICAtMSAgMXMgICAxICAtMC41MDAwMDAwRSswMA0KICAyICAgMSAgMnAgICAzICAtMC4yNTAw
MDAwRSswMA0K
```

Every whitespace byte in the payload is discarded before it is decoded, so how
the file was wrapped, indented or line ended cannot reach the bytes. That is the
whole property, and `TestHowTheFileIsWrappedCannotReachTheBytes` in
`internal/fixture` asserts it rather than leaving it as a claim.

The four fields are all required and no other field is permitted. A misspelt
one is refused rather than ignored, because otherwise it is a provenance note
that silently is not there.

- `Code` is the program the bytes came from, or `hand-written` where no program
  produced them.
- `Version` is the version of that program.
- `Case` is what it was asked to compute.
- `Kept` says how much of the original file was retained. A reader meeting a
  table that stops after two lines needs to know it was cut here rather than
  there.

Where a field does not apply, write `not applicable` rather than leaving it
empty. An empty field reads as answered and is refused for that reason.

## Adding one

Render the payload rather than assembling it by hand:

```go
fmt.Print(fixture.Encode(raw))
```

Read it back with `fixture.Load`, which is the only thing that reads a fixture
and is the same code the gate judges with, so a fixture that loads in a test is
a fixture the gate accepts.

```go
f, err := fixture.Load(filepath.Join("testdata", "levels.fixture"))
```

Assert on `f.Bytes` and not on something parsed out of it. A parsed result is
exactly what survives the damage this convention exists to prevent, so a test
that only checks parsed values would stay green through the failure.

## What the check refuses

Every file under a `testdata` directory, no exceptions, and a file named
`*.fixture` anywhere else because nothing would look for it there. The rule is
total rather than a list of exemptions: whether a particular file's exact bytes
matter is a judgement, and a checkout does not make it.

## What it does not do

It says nothing about whether the recorded bytes are the right bytes. A fixture
carrying invented numbers in the correct layout and a fixture carrying a real
run are indistinguishable to every check here, and the `Code` field is the only
thing that tells them apart. It is a note that somebody wrote, not a measurement.

Whether recorded output from every participating code may be committed at all is
not settled. It is entry 3 of the maintainer decisions issue, #1, and the
fallback if the answer is no is synthetic fixtures in the same layout carrying
invented numbers.
