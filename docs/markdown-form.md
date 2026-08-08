# The form documentation is written in

Three legs of `go run ./gate` read the Markdown in this tree: `documentation
form`, `documentation links` and `documented paths`. This file is what they read
it against, so the form is decided once here rather than argued about on each
change.

## The form

Judged after normalising CRLF to LF, and the leg says when it normalised. A
checkout with `core.autocrlf` set hands the working tree carriage returns on
every line of every document, and a leg reporting that as a fault in the writing
would say nothing about the writing. What the bytes in a checkout are is a
separate question and is not this leg's.

Outside a fenced code block:

- A line does not end in whitespace. It is invisible in the editor that made it
  and it survives into every diff afterwards.
- A line holds no tab. A tab's width is a setting on the machine reading it
  rather than a property of the file.
- No two blank lines in a row. One separates; the second says nothing the first
  did not.
- A heading is one to six hashes, one space, and a title. Not two spaces, not a
  closing run of hashes, and not the underlined form, so that every heading in
  the tree is found by the same search.

Over the whole file:

- It does not start with a blank line, it ends with exactly one newline, and
  every fence it opens is closed.

Inside a fenced block, nothing above applies. A fence holds bytes quoted from
somewhere else, usually a command and what it printed, and a rule that tidied
those would be rewriting the evidence the document is carrying.

## What the leg does not do

It names the file, the line and the rule, and it rewrites nothing. There is no
formatter to run, so a repair is an edit somebody makes and reads. That is the
cost of not taking a Markdown formatter as this module's first dependency, which
`docs/dependencies.md` is where it would have to be argued.

## Links

`documentation links` resolves every link that points inside this tree, from
both `[text](target)` and `[label]: target`, and refuses one that resolves to
nothing.

A link with a scheme is not resolved here. Somebody else's server being down is
not a reason to refuse a change in this repository, and it is a reason to know,
so external links are checked on a schedule by `tools/externallinks` instead.

A fragment is not resolved either. Nothing here reads a document's heading
structure, so a link to a section that has since been renamed passes. That is
the bound on the leg rather than something it hides.

## Paths named in prose

`documented paths` reads every path a document names, and not only the ones
inside link syntax. Documents here name files in plain sentences as often as in
links, and a check reading only the marked-up half would pass a document whose
sentences describe a tree that is not this one.

The scope is every Markdown file in the tree, including the ones at the
repository root. A document that imposes a rule is inside the reach of the
mechanism that reads it, which a scope stopping at `docs/` would not have
managed.

What counts as a path is deliberately narrow. It holds a slash, no whitespace,
no `@` and no colon, and its first segment is an entry at the repository root.
The last of those is what keeps the leg off the things a document about this
project has to name and which are not files here: a standard library import
path, a GOOS and GOARCH pair, the layout inside a bundle that a run writes. All
of those are two words with a slash between them, and no reading of the string
separates them from a relative path.

What it costs is that a path under a top level directory that does not exist is
not read at all, so renaming a top level directory stops the leg reading the
paths under the old name rather than refusing them. The links leg still resolves
whatever is written as a link, and that is the half of the cover this bound does
not remove.

A document that has to name a path it does not intend to resolve writes it
without a slash, or writes it as prose the rule above does not reach. A rule
with no way out is one somebody eventually switches off.
