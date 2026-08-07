# 0001. The harness is written in Go

Number: 0001
Title: The harness is written in Go
Status: accepted
Date: 2026-08-07

## What was decided

The harness is written in Go. That covers the command line tool, the case loader,
the container runner, the readers, the common data model, the identification and
comparison code, the bundle writer and the report renderer. Every part of the
thing an operator installs is one binary built from this tree.

The minimum version the project builds against is Go 1.24, and it is declared in
one place rather than restated. That place is the module file, which the
scaffolding milestone owns:

    git show origin/main:go.mod
    module github.com/iderex/gegenprobe

    go 1.24

A version older than the directive is not supported and is not tested. Raising
the floor is a change to `go.mod` and to this record's successor, not a change to
this record.

This decision does not cover the parameter fit component. That component is
started as a separate process behind the same contract a code is started under,
and its language is chosen again from its own requirements when its milestone
begins. 0013 is where that is written down.

## Why

The thing an operator installs has to be one file that runs. This project's whole
claim is reproducibility, and a tool whose own installation needs a scientific
runtime, a package resolver and a working compiler toolchain has spent its
credibility before the first case runs. A statically linked binary with no
runtime removes a class of failure the project exists to remove.

The harness's real work is process control. Start a container, feed it input,
collect files, apply a timeout, cancel cleanly, capture what actually happened.
The Go standard library covers that directly, so the dependency list stays short
enough that somebody can read all of it in an afternoon, which is a precondition
of the supply chain work in the quality milestone rather than a nicety.

Parsing fixed format Fortran output is byte work over text, and Go does byte work
over text plainly. A table driven reader suite over recorded fixtures runs in
seconds, and a suite that runs in seconds is one people keep adding to.

Byte stability is a requirement everywhere in this project, and 0007 makes it a
property of the bundle. Go makes it reachable without heroics: map iteration
order is randomised loudly rather than stable by accident, so a route that
depends on it fails early and visibly instead of on somebody else's machine.

Neutrality matters here in a way it would not on another project. This is a bench
that publishes numbers about other people's codes. Writing it in the language of
one of the participants would make that participant the home team in appearance
if not in fact. On its own that argument decides nothing, and what it does is
remove the one candidate that would otherwise be attractive.

## What was rejected

Python. The field's common language, FAC already ships a Python interface, and
the scientific stack is genuinely better at the numerics. It loses on the thing
that matters most here, which is what has to be working on the operator's machine
before anything runs at all. An environment that resolves differently on two
machines is a reproducibility defect in a reproducibility tool. The numerics
advantage is real, and it is exactly why 0013 refuses to bind the fit component
to this decision.

Julia. Same language as JAC, good numerics, and it would make one participant's
integration trivial. Rejected for the neutrality reason above, and because it
puts a large runtime between the operator and a tool whose job is mostly file
handling and process supervision.

Rust. It satisfies the single binary argument and the parsing argument as well as
Go does, and on memory safety it is ahead. Rejected on build friction and on the
size of the pool of people who could review a change here. For a small scientific
tool that will be read by more people than write it, reviewability weighs more
than raw language quality.

Shell scripts driving the codes directly. This is how the field already does it,
and reproducing that shape would be reproducing the problem the project is about.
It also has no place to put a reader, which is where most of the work is.

## What this costs

Go's numerical ecosystem is thin. Nonlinear least squares over a rough landscape
and global optimisation, which the fit component needs, are not available here at
the quality that problem demands. The plan accepts that in the open rather than
pretending otherwise, and the fit runs as a separate component behind the same
process contract the codes use.

Go is not a language most people in atomic physics write, so a contributor from
the field meets a barrier that Python would not have raised. What softens it is
that the parts a physicist is most likely to want to change, the case files, the
container recipes and the tolerances, are data rather than code. It is a
softening and not a removal, and a reader who wants to change a matching rule
still has to write Go.

Fixing a minimum version at all is a cost paid on the machines that have an older
toolchain, which on managed institutional desktops is a real population. The
floor buys generics in the reader contract and the newer standard library
routines the runner uses, and it is the number the module file already carries
rather than one chosen here to be comfortable.

Nothing in this record is enforced. No check in this tree refuses a second
language appearing in it, and the module file's directive is a build constraint
rather than a rule about what may be added beside it. The repository invariant
lint planned in the quality milestone is where that would be refused, and until
it lands this record is a description.
