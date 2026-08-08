# The dependency budget

Direct dependencies: 0

Record 0001 argues for a small dependency surface and says the list has to stay
readable in an afternoon. An argument with no number behind it is worth nothing,
so the number is here and the `dependencies` leg of `go run ./gate` refuses a
`go.mod` that disagrees with it. Adding a dependency is therefore an edit to this
file in the same change, which is what makes it a decision with a reason rather
than a convenience.

## The reasons

One section per direct dependency, headed by its module path, saying what it is
for and what taking it costs. The leg refuses a direct requirement with no
section here, and a section naming a module the tree does not require.

There are none today. The tree carries no `go.sum`, because nothing has been
downloaded to record a checksum for:

```
go mod verify
all modules verified
```

That output is what verifying an empty set looks like, and the leg says so rather
than letting a green line stand for a verification that examined nothing.

## What a new one has to say

What it does that the standard library does not. What the tree carries instead
today, and why that is worse. What the cost is if the project is abandoned, since
this repository is meant to be readable by somebody in a regulated environment
who has to account for what they are running.

Whether the answer is good is what the review is for. What this file and the leg
make checkable is that the question was answered at all.
