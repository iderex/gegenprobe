# Which package may import which

The `package boundaries` leg of `go run ./gate` reads the import graph of this
module and refuses an edge this file does not permit. The permitted direction is
data here rather than a diagram in a document, so an edge nobody argued for is a
red run rather than a paragraph somebody stopped believing.

This is the structural half of what `docs/dependencies.md` does for the surface
outside the module. That file answers which foreign code this tree carries at
all; this one answers which parts of this tree may reach which other parts.

## How to read an entry

One section per package, named by its path relative to the module, with `.` for
the command at the root. Each carries the decision the boundary comes from and
two lists.

`May-import` is what the package's own source may import from this module.
`May-import-in-tests` is what its test files may import in addition. They are
separate because a package that reads nothing and a package whose tests read a
fixture reader are different statements, and collapsing them would let the first
quietly become the second.

Both lists are exact and neither is a prefix rule. `nothing` is a permitted value
and means what it says. A package absent from this file is refused, so a new one
has to be placed rather than defaulted into whatever it happens to import first.

The leg reads the graph the toolchain reports rather than the import lines in the
source, so an edge that arrives through a rename or a move is seen the same way
as one somebody typed.

## The entries

## .

Decision: 0001
May-import: internal/cli
May-import-in-tests: nothing

The command an operator installs. 0001 fixes that it is one binary built from
this tree, and the command layer holds no logic: it reaches the command line
package and nothing else, so anything it grows has to be placed somewhere a test
can reach without building a binary.

## internal/cli

Decision: 0001
May-import: internal/version
May-import-in-tests: nothing

## internal/version

Decision: 0001
May-import: nothing
May-import-in-tests: nothing

The version is what every other part is allowed to depend on and it depends on
nothing, which is what keeps it importable from anywhere without carrying
anything along with it.

## internal/fixture

Decision: 0009
May-import: nothing
May-import-in-tests: nothing

The only thing that reads a fixture. 0009 permits a gate tier test to read files
under a `testdata` directory, and this package is how that is done, so it has to
be reachable from every tier without pulling any of the tree behind it.

## internal/boundary

Decision: 0009
May-import: nothing
May-import-in-tests: internal/fixture

It judges an import graph handed to it against a declaration handed to it, so
the judging is testable over recorded graphs and only its caller has to ask the
toolchain what the real graph is. Reading anything of this tree would make it a
participant in the thing it judges.

## internal/commit

Decision: 0009
May-import: nothing
May-import-in-tests: internal/fixture

It judges commit messages handed to it and reads nothing itself, which is what
lets the same judgement serve the leg and the command without either holding a
copy of it. Its tests read recorded ranges, which is the fixture reader and
nothing more.

## internal/model

Decision: 0004
May-import: nothing
May-import-in-tests: internal/fixture

The types every reader produces and every later stage consumes, and the schema
generated from them. 0004 makes it the thing the readers, the comparison and the
renderer all agree through, so it depends on none of them: an edge from here into
a reader would make the shared vocabulary a participant in one code's output.

Its tests read a stored bundle from a format this build does not read, which is
the fixture reader and nothing more.

## internal/golden

Decision: 0009
May-import: nothing
May-import-in-tests: nothing

It compares an artefact a producer wrote against the copy this repository holds,
and it is handed both. Reading anything of this tree would make it a participant
in the comparison, which is the same reason `internal/boundary` reads nothing.

It is the one package here whose own source imports `testing`, because the
assertion and the flag that rewrites a golden belong beside the comparison they
use rather than in each caller. That is outside this file's subject, which is
edges inside this module, and it is recorded here because it is the thing a
reader of this entry would otherwise have to discover.

## gate

Decision: 0009
May-import: internal/boundary, internal/commit, internal/fixture
May-import-in-tests: internal/fixture

The single check command. It may reach into this module to reuse judging that
already exists, and nothing may reach back into it: it is not part of the binary
an operator installs, and no entry above lists it.

## tools/commithygiene

Decision: 0001
May-import: internal/commit
May-import-in-tests: nothing

## tools/decisionindex

Decision: 0000
May-import: nothing
May-import-in-tests: internal/golden

The index over the decision records is generated rather than typed, which is
0000's own requirement, and the generator reads the records and nothing else.

Its tests reach the golden helper, and only its tests. The command writes the
index; what compares the committed one against a fresh render is an assertion in
the suite, so the generator does not carry a comparison it would then be the only
caller of.

## tools/externallinks

Decision: 0001
May-import: nothing
May-import-in-tests: nothing

## What this leg does not do

It says nothing about whether a permitted edge is a good idea. A list somebody
widened without thinking is indistinguishable from one somebody argued for, and
the review is where that is caught.

It says nothing about the standard library or about anything outside this module.
`docs/dependencies.md` holds that surface and the `dependencies` leg refuses a
change to it.

It does not read the schema. The other half of the conformance work asks that
every type a bundle carries be covered by the schema and every schema field
correspond to a type field, checked in both directions. That comparison is made,
and it is made in `internal/model`, over the type set the schema is generated
from, by that package's own suite in the gate tier. It is deliberately not a leg
here: this leg reads the graph the toolchain reports and that one reads a type
set, which are two sources, and one leg named for conformance that reads two
unrelated things is the shape that later gets split. What each covers is said in
the run by two lines rather than by one, and both come from 0004, which makes the
model the thing every reader and every later stage agrees through.
