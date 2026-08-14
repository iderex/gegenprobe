<!--
Three questions, and the third one only applies to a change that adds a check.
Delete the section that does not apply; an empty one says nothing.
-->

## What changed

<!-- What this does, in the terms somebody reading the tree in a year would use. -->

## What failure it prevents

<!--
Not what it improves. What goes wrong without it, and how you know that failure
is real. If it has already happened, say where.
-->

## Where the proof that the check bites lives

<!--
For a change that adds a check, and required for one.

A check nobody has watched fail is an explanation of a rule rather than a rule.
Name the fixture that trips it and the test that asserts the trip, and say what
the fixture changes relative to a passing one. One line is the target: a
near-miss that could not have failed proves less than one that nearly did.

Then say what happens when the check is disabled and the same fixture is run.
If it does not go green, the fixture is not proving what it claims.

Delete this section if the change adds no check.
-->

## Evidence

<!--
Every number here carries the command that produced it, run at the commit being
pushed and never in your working tree. Where a claim cannot be backed by a
command, write it as a claim and say so.

Paste the commands and their output.
-->

## Issue

<!-- Closes #NNN, or Refs #NNN where it does not close it. -->
