# 0012. Everything stays on the host unless the operator deliberately publishes

Number: 0012
Title: Everything stays on the host unless the operator deliberately publishes
Status: accepted
Date: 2026-08-07

## What was decided

Everything stays on the host. The harness sends nothing anywhere by itself. There
is no telemetry, no usage counting, no update check, no crash reporting and no
error submission, and none of these is added later behind a default on switch.

The only network access at run time is fetching container base images, which
happens through the engine the operator configured, to registries the operator
named, and only when an image is missing.

Personal data reaches the artefacts through three routes and each is handled.
Absolute paths reveal a home directory and therefore a user name. Host names and
user names appear in container and process metadata. A case file may carry an
author name or an email if the operator put one there.

The run record keeps what reproducibility needs and nothing about the person.
Paths inside the bundle are relative to the run directory. Where an absolute path
is genuinely needed for diagnosis it goes into the raw output area, which is
outside the checksummed bundle and is not part of what gets shared.

Publishing is a separate, deliberate act, invoked by its own command, never a
side effect of running a case. That command prepares a shareable bundle, redacts
host name, user name and absolute paths by default, and prints exactly what it
removed and what remains before it writes anything. Federating results to any
shared collection is opt in per run, and there is no configuration that makes it
automatic.

If a case file carries an author identity, that field travels only when the
operator publishes, and it is listed in the pre publication report like
everything else.

### What is redacted by default

- Host name, wherever it appears: container metadata, process metadata, engine
  output, and any line a code printed that echoed it.
- User name, user id and group id, from the same places.
- The home directory string, and every absolute path, replaced by a path relative
  to the run directory or by a marker where no relative form exists.
- Operator supplied source paths, which name a directory somebody chose and often
  a project.
- The environment captured in the run record, except for an allowlist of
  variables that affect the numerical result. Anything outside that list is
  removed rather than reviewed.
- The container engine socket path, which carries a user id on the usual
  installations.
- The registry host an image came from. A private registry hostname names an
  institution. The image digest stays, and it is the part that carries the
  provenance.
- Any IP or hardware address that appeared in captured metadata.
- The case file's author name and email, where the case carried them.

### What is not redacted, and why each one stays

- **The canonical case and its identity hash.** Without them nothing in the
  bundle can be checked, and they describe the question rather than the person
  asking it. Where an operator put a name into the case, that is the author field
  above and it is removed; the rest of the case is physics.
- **The case name or title the operator chose.** It is how a reader refers to the
  case and how two bundles are told apart. It is also free text somebody chose,
  so it can name a project or a group. It stays, and it is listed in the pre
  publication report so the operator sees it before deciding.
- **Image digests, recipe identifiers and code versions.** They name software,
  not people, and they are the whole provenance of a number.
- **The harness version and commit.** Same reason.
- **Container engine name and version.** A numerical result can depend on it, and
  it names software.
- **CPU architecture and operating system family**, for example `linux/amd64`.
  These name a class of machine rather than a machine, and floating point results
  can depend on them.
- **Run start time and duration.** They are needed to read a timeout or a
  resource limit that fired, and they say nothing about who ran it. They are
  listed in the report anyway, because an operator may not want them and the
  report is where that choice is made.
- **Every physical value, its significance, and the tolerances in force.** This
  is the artefact.

The list above is the default. The pre publication report prints the actual
decision for every field in the bundle at hand, so a field this record did not
anticipate is shown rather than assumed.

### The wording that has to appear

This wording, or wording that says the same thing without weakening it, appears
in `README.md` and in the operator documentation:

> This tool runs on your machine and sends nothing anywhere on its own. It has no
> telemetry, no usage counting, no update check and no crash reporting. The only
> network access it makes is fetching container images, through the engine you
> configured, from the registries you named, and only when an image is missing.
>
> Publishing is a separate command and never happens as a side effect of running
> a case. Before it writes anything it prints what it will remove and what it will
> keep.
>
> Redaction removes host names, user names and absolute paths. It does not make a
> bundle anonymous. A case name, a configuration label or a directory you chose
> can identify you or your institution. The report shows you what is being
> published; it does not promise that what remains is anonymous.

The last paragraph is the load bearing one and it is not softened. Every claim
above is one the project can defend from its own source: no outbound call
exists, publishing is a distinct command, and the report is printed before the
write. The thing the project cannot defend is anonymity, so it does not claim it.
A future edit that turns that paragraph into an assurance is removing the only
honest sentence in the passage.

## Why

Comparison results are the kind of artefact that ends up attached to a paper or
dropped into a shared folder, and by then nobody rereads what is inside them. The
moment to be careful is when the bundle is made, not when it is sent.

Redaction as a default with an explicit report, rather than as an option, is the
arrangement that survives a hurried user. A flag that has to be remembered is a
flag that gets forgotten on the one bundle that mattered.

Telemetry is refused outright rather than made opt in because this project's
users include people on hospital, government and laboratory networks where an
unexpected outbound connection from a research tool is a genuine incident. Being
able to say the tool never initiates one is worth more than any usage statistic
it could gather.

Saying it in the documentation is part of the decision rather than a follow up.
An operator deciding whether they may run this on a managed machine needs the
answer before they install it, in the README, not in a privacy page nobody links.

Naming what is not redacted, one field at a time with its reason, is what stops
the default from quietly growing. A list of removals alone invites the reading
that everything else was considered and found harmless, and most of it was never
considered at all.

## What was rejected

Anonymous usage telemetry with an opt out. Standard practice, useful to
maintainers, and it makes the paragraph above impossible to write truthfully.

Redaction as a flag on the publish command. One word cheaper, and it fails in
exactly the case that matters.

Redacting at run time rather than at publish time. Would produce bundles that are
safe by construction, and it would strip the diagnostic detail an operator needs
on their own machine, which is where the tool is least dangerous.

Redacting the case name as well, on the grounds that it is free text. It would
make two bundles from the same operator hard to tell apart, and the honest
handling of free text somebody chose is to show it to them before it leaves,
which the report does.

Describing the result as anonymised in the documentation. Shorter, expected by
readers, and untrue.

## What this costs

The project learns nothing about how it is used, so decisions about what to
improve rest on what people report rather than on measurement.

Redaction is never complete. A configuration name, a project directory or a case
name can identify a person or an institution, and the pre publication report
shows what is being published rather than promising it is anonymous. The
documentation has to say that in those words.

The environment allowlist has to be maintained, and a variable that affects a
numerical result and is not on the list will be removed from the run record,
which costs reproducibility in exactly the case where it is hardest to notice.
Erring towards removal is the intended direction and it is still a cost.

Keeping the case name means a bundle can carry an institution's name into a
public collection, and the only defence is a report somebody has to read.
