# Decision records

One file per decision, in the format fixed by
[0000](0000-decision-records.md).

This index is generated. Edit a record and regenerate it rather than
editing this file:

    go run ./tools/decisionindex

| Number | Title | Status | Date |
| --- | --- | --- | --- |
| [0000](0000-decision-records.md) | Decision records | accepted | 2026-08-07 |
| [0001](0001-the-harness-is-written-in-go.md) | The harness is written in Go | accepted | 2026-08-07 |
| [0002](0002-the-case-file.md) | The case file is YAML, canonicalised to JSON before anything reads it | accepted | 2026-08-07 |
| [0003](0003-recipes-not-images.md) | This repository ships container recipes, never images and never the codes | accepted | 2026-08-07 |
| [0004](0004-the-common-data-model.md) | The common data model, and the reciprocal centimetre as the canonical unit | accepted | 2026-08-07 |
| [0005](0005-identification-is-a-separate-step.md) | Identifying the same level across codes is a separate step that is allowed to fail | accepted | 2026-08-07 |
| [0006](0006-what-agreement-means.md) | What the harness means by agreement, and what it refuses to conclude | accepted | 2026-08-07 |
| [0007](0007-bundle-and-report.md) | The results bundle and the rendered report are separate artefacts | accepted | 2026-08-07 |
| [0008](0008-determinism-and-significant-digits.md) | Determinism in the harness, and never printing more digits than a code produced | accepted | 2026-08-07 |
| [0009](0009-three-test-tiers.md) | Three test tiers, and the gate tier needs no display, no elevation and no network | accepted | 2026-08-07 |
| [0010](0010-jac-under-the-same-contract.md) | JAC runs under the same contract as the Fortran codes | accepted | 2026-08-07 |
| [0011](0011-four-states-for-a-missing-value.md) | Four states for a value that is not there | accepted | 2026-08-07 |
| [0012](0012-everything-stays-on-the-host.md) | Everything stays on the host unless the operator deliberately publishes | accepted | 2026-08-07 |
| [0013](0013-the-fit-is-a-separate-component.md) | The fit component inherits no language from the harness | accepted | 2026-08-07 |
| [0014](0014-the-fit-component-is-python-with-numpy-and-scipy.md) | The fit component is Python with NumPy and SciPy | accepted | 2026-08-09 |
