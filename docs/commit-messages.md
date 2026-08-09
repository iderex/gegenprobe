# What a commit message may say, and which characters it may say it in

Two checks read the commits a change is made of. `internal/commit` judges them,
`tools/commithygiene` is the command that runs it over a range, and the leg named
`commit hygiene` in `go run ./gate` runs the same command over the range between
the default branch and the work. This file is the one place the rules are
written down, so widening either of them is an edit somebody reads here rather
than a constant somebody changes in passing.

## The subject carries the issue

Every commit subject names the issue the commit belongs to, as `#` followed by
digits, anywhere in the first line.

The subject is the line that survives. A log listing, a blame view, a release
note and a bisect run all show it and show nothing else, so a link that lives
only in the body is a link that is absent exactly where somebody is looking for
it.

Two kinds of commit are exempt.

A merge commit, meaning one with more than one parent. Its subject is written by
the forge and its issue reference is in the pull request it names.

A commit whose author is one of the bots this repository has decided to trust:

Exempt-Author: dependabot[bot]
Exempt-Author: github-actions[bot]

The list is exact names rather than a suffix rule. `[bot]` at the end of a name
is a convention on the forge and not a fact about the account, so a rule reading
it would exempt anybody who chose the name.

### What this check does to a contribution from outside

A contribution from outside this repository cannot know an issue number before
the issue exists, and asking a contributor to guess one is asking them to write
something wrong. So where the head of a pull request is not a branch of this
repository, the subject rule reports and does not fail. The linkage is added
when the contribution is handled.

The character rule is not relaxed for anybody. It is about what the message is
made of rather than about what it knows.

## The characters

An allowlist, not a list of forbidden characters. A list of the characters known
to be dangerous refuses the attack somebody has already written up; a list of the
characters this repository actually uses refuses the one nobody has thought of
yet. That is the reasoning behind the Trojan Source work and the Unicode security
guidance the existing `Reject Trojan Source Unicode` check comes from, and this
check is the same reasoning applied to the half that check cannot reach: it reads
files in the tree, and a commit message is not a file in the tree.

A commit message is also the artefact that cannot be corrected. A bad line in a
file is fixed by a later commit. A bad line in a message is fixed only by
rewriting history, which on a branch other people have pulled is not a repair at
all.

What a message may hold:

Allowed: U+000A
Allowed: U+0020-U+007E

That is the line feed and the printable ASCII range. Nothing else, which means no
tab, no carriage return, no non-breaking space, no typographic dash and no
character from any script beyond the one this repository writes in.

### What was measured

The whole of the message history on the default branch, at the commit this file
landed against:

```
$ git log --format='%B' origin/main | python -c "
import sys
s = sys.stdin.buffer.read().decode('utf-8')
cps = sorted(set(ord(c) for c in s))
print('distinct code points:', len(cps))
print('outside U+0020-U+007E and the line feed:',
      [hex(c) for c in cps if c > 0x7E or (c < 0x20 and c != 0x0A)])
print('lowest:', hex(cps[0]), 'highest:', hex(cps[-1]))
"
distinct code points: 83
outside U+0020-U+007E and the line feed: []
lowest: 0xa highest: 0x7a
```

Eighty-three code points, the lowest the line feed and the highest `z`. Nothing
in the history is outside what is allowed above, and the allowance is wider than
the measurement by the characters between `z` and `~` that nobody has needed yet.

The allowance is deliberately not narrowed to exactly what was measured. A rule
that refuses `{` because no message has used one is a rule that fails on the
first message quoting a Go literal, and a check that fails for a reason nobody
can defend is a check that gets switched off. The line is drawn at the end of
printable ASCII because that is a boundary with a reason behind it rather than a
boundary drawn around a sample.

### Widening it

Add an `Allowed:` line here, in the same form, and say in the same change what is
being admitted and why. That is the whole mechanism, and the point of it is that
the diff shows a rule moving rather than a regular expression growing a bracket.

## What these checks do not do

They say nothing about whether the message is any good. Whether it states what
changed, whether it says what failure the change prevents, and whether it carries
one topic rather than two are judgements, and no reading of a commit makes them.

They say nothing about the issue the subject names. A number that refers to a
closed issue, to an issue in another repository, or to nothing at all passes, and
the reference in the subject of a commit cannot be checked against the tracker
without reaching the network, which the tier this leg runs in may not do.

The subject rule is a change of practice rather than a continuation of one. At
the commit this file landed against, no commit in the history satisfies it:

```
$ git log --no-merges --format='%s' origin/main | wc -l
20
$ git log --no-merges --format='%s' origin/main | grep -cE '#[0-9]+'
0
```

Twenty commits, none of them naming an issue in the subject, because the
reference has been going in the body and in the pull request instead. The check
reads the commits a change is made of and never the history behind it, so nothing
already landed is reddened by this. What it means is that the practice starts
here, and a reader who greps the log for an issue number will find the older half
of the history silent.
