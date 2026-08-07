# 0002. The case file is YAML, canonicalised to JSON before anything reads it

Number: 0002
Title: The case file is YAML, canonicalised to JSON before anything reads it
Status: accepted
Date: 2026-08-07

## What was decided

A case is one file in YAML 1.2, with a required version field, validated against
a JSON Schema shipped in this repository.

Before anything reads a case, the loader converts it to a canonical JSON form:
keys sorted, defaults written out, units normalised, no comments and no aliases.
Everything after the loader reads that canonical form and never the YAML. The
SHA-256 of the canonical form is the case identity and appears in every artefact
the run produces.

The case file names physics, not code flags. Element, charge state, reference
configurations, the configuration set to include, the properties requested.
Anything that exists in one code and has no counterpart in another lives in a
clearly separated per code section, and a case that sets a per code knob for a
code not in the run is refused rather than ignored.

Unknown fields are refused, not skipped.

### The versioning rule

The case file carries `version`, a single positive integer. It is the version of
the case schema, not of this repository and not of the harness.

The number is bumped by any change that can make a previously valid case invalid,
change what a field means, or move the canonical bytes of a case that did not
itself change. Adding an optional field with a default bumps it too, because
canonicalisation writes defaults out, so a new default changes the identity of
every case that did not set it. There is no minor number, because there is no
class of schema change here that leaves both validity and identity untouched.

A loader that meets a version it does not know refuses the case. It names the
version it read and the versions it supports, and it does not guess, does not
fall back to the nearest one, and does not read the file far enough to act on
anything else in it. A case from the future is the one shape where a partial read
is most tempting and least defensible.

Every version the project has released stays readable for at least twenty four
months after the release that replaced it. Dropping one is a decision in its own
right and gets its own record. A case pinned to a dropped version is refused with
a message naming the last release that could read it, so the operator has
somewhere to go rather than a rejection.

### The canonicalisation rules

These are written to the byte, because the identity hash is worthless if two
implementations disagree about them.

Reading:

- The parser reads YAML 1.2 with the core schema and nothing else. Duplicate
  keys, anchors, aliases, merge keys, non string keys, explicit tags and
  directives other than `%YAML 1.2` are refused rather than resolved.
- Comments are read and discarded. They never reach the canonical form.
- A null, in any of YAML's spellings, is refused wherever it appears. Absence is
  how a case says nothing, and a key present with no value is a typing accident
  rather than a statement.
- A boolean field accepts `true` and `false` only. YAML 1.2's core schema already
  reads `yes`, `no`, `on` and `off` as strings, so the loader refuses them by
  type rather than reinterpreting them, and says so in the message, because that
  is the trap an operator coming from YAML 1.1 will hit.
- A field whose schema type is a string is refused if it was written unquoted and
  the core schema resolved it to a number or a boolean. This is the sexagesimal
  and version-number trap, and refusing is the only handling that cannot be
  misread.

Writing:

- The output is JSON per RFC 8259, encoded UTF-8, with no byte order mark.
- No insignificant whitespace anywhere. No space after `:` or `,`, no newlines,
  no indentation. The whole document is one line.
- Object keys are sorted ascending by Unicode code point of the key string. Keys
  are compared as sequences of code points, not by locale and not
  case-insensitively.
- Every field the schema gives a default is present, carrying either the
  operator's value or the default.
- Arrays keep the order the operator wrote, because order is meaningful in a
  configuration list. The exception is a field the schema marks as a set, and
  those are sorted ascending by the canonical encoding of their elements. Which
  fields are sets is stated in the schema and nowhere else.
- Strings escape `"` and `\` and the control characters below U+0020, the latter
  as `\b`, `\f`, `\n`, `\r`, `\t` where a short form exists and `\u00xx` with
  lowercase hex otherwise. `/` is never escaped. Characters above U+007F are
  written literally as UTF-8 and never as `\u` escapes.
- Numbers are never converted to binary floating point on this path. The
  operator's literal is normalised as text: a leading `+` is dropped, leading
  zeros in the integer part are reduced to one digit, a digit is required either
  side of a `.`, an exponent is written `e` with an explicit sign and no leading
  zeros in the exponent digits, and a value with no fractional part and no
  exponent is written as an integer. No rounding and no reformatting beyond that.
- Units are normalised before the number is written, and the unit suffix does not
  survive into the canonical form. Which unit each quantity is normalised to is
  the common data model's decision, 0004, and is not restated here.
- The document ends at its closing brace. There is no trailing newline, and the
  hash is taken over exactly those bytes.

The case identity is the SHA-256 of that byte sequence, written as lowercase
hexadecimal, in full, never truncated in an artefact.

## Why

An operator writes this by hand and reads it again a year later, which rules out
anything uncomfortable to type or to comment. That is the whole argument for YAML
and it is enough.

The canonical form exists because a comparison bench cannot afford ambiguity
about what was asked. Two YAML files differing in key order, in quoting, or in
whether a default was spelled out are the same case, and they have to produce the
same identity or nothing downstream can be cached, diffed or trusted.
Canonicalising once, at the edge, means exactly one piece of code holds an
opinion about what a case means.

Refusing unknown fields matters more here than in most tools. A misspelled key
that is silently ignored produces a run that looks successful and answers a
different question than the operator asked, and in this domain that difference
surfaces later as a method disagreement in a published table.

Separating physics from per code knobs is the load bearing rule. If a knob that
changes the physics can be set for one code and not for the others, the bench is
comparing two different calculations and reporting the difference as a method
difference. Keeping those knobs visibly apart is what makes that mistake hard to
make by accident.

Keeping numbers as text through canonicalisation is the same argument one level
down. A loader that parses `1.30` into a float and prints it back gets `1.3`, and
two cases that differ only in a trailing zero would then share an identity while
the significant digits rule in 0008 treats them as different statements about
precision.

## What was rejected

Bare JSON as the authored format. Unambiguous, and unpleasant to write by hand
for nested configuration lists. It survives as the canonical form, which is where
its properties are actually wanted.

TOML. Fine for flat configuration, awkward for the nested lists this needs.

A purpose built input language. The existing codes each have one, they are a
large part of why the field is hard to enter, and adding another would be an odd
way to fix that.

Accepting each code's native input directly. That removes the translation layer
and with it the point, since the project's claim is that one description drives
all of them.

A major and minor schema version. The minor number would have to mean a change
that leaves both validity and canonical bytes untouched, and writing defaults out
means no such change exists here.

Canonicalising through a generic JSON round trip in the implementation language.
Every such round trip carries that language's number formatting, and the identity
would then be a property of the toolchain rather than of the case.

## What this costs

Two formats to keep in step, and a schema that has to be versioned carefully,
because a case file written today should still run in two years or fail with a
clear statement of why not.

YAML has sharp edges, in particular around unquoted values that look like
numbers, booleans or sexagesimals. The loader has to pin the parser's behaviour
and test the known traps explicitly, and the schema work is not finished until it
does.

Writing defaults into the canonical form means the identity of a case moves when
a default moves, so a schema bump invalidates every cached result. That is the
correct behaviour and it is still a cost, paid whenever the schema changes.

Normalising numbers as text rather than as numbers means the loader carries its
own small piece of lexing that a library would otherwise have done, and that code
has to be tested against the literals YAML permits rather than against the ones
anybody writes on purpose.

Supporting every released schema version for two years means the loader
accumulates readers it cannot delete on the schedule that would suit it.
