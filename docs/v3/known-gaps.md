# v3 Known Gaps

Verified, unfixed defects. Each entry records how it was checked so a later
reader can confirm it still holds instead of trusting this file.

## Open

**`PolicyHash` depends on the checkout's line endings.** The hash is sha256
over the policy file's raw bytes, so a checkout that converts `\n` to `\r\n`
changes a policy's identity without changing the policy. The same file in this
repository hashes to `ee8bd24f…` as git stores it and `6bbdbdfb…` in a Windows
working tree with `core.autocrlf=true`.

How it was found: `TestFuzzSubjectMatchesGenerator` passed locally and failed in
CI on the rc.7 tag. The committed fuzz subject carried a hash of CRLF bytes;
the Linux runner, checking out LF, computed a different one.

What is fixed: `.gitattributes` pins this repository to LF, so generation here
is reproducible and CI and a developer machine agree.

What is not: a consumer generating a policy on Windows with `autocrlf=true`
still gets a different `PolicyHash` than their Linux CI does from the same
committed file. Their "generated code is current" check then fails for a reason
that has nothing to do with their policy — or worse, someone silences it.

Why it is not simply fixed here: normalizing line endings inside the hash would
make it platform-stable, but it changes every `PolicyHash` ever recorded, and
`AGENTS.md` states the hash covers the file's bytes. That is a decision about
the audit contract, not a bug fix. Until it is made, a consumer should commit a
`.gitattributes` of their own.

Note that `TestGenerationIsByteStableAcrossRuns` does not cover this: it proves
generation is stable across runs in one process, not across platforms.

## Closed

Last verified at `v3.0.0-rc.7` on 2026-08-07. All
seven entries opened during the rc.5/rc.6 work were fixed the same day:

- the conformance recipe pointing at a test name nothing matches;
- `repo-hygiene.sh` being invoked by no workflow and no script;
- `check -format json` writing its result to stderr;
- the unpinned error-code wire strings;
- the public API inventory listing four of ten packages;
- no snapshot field responding to a policy's facts or predicates;
- `generated-policy-metadata.schema.json` describing a document nothing
  produced.

The last four are now enforced by tests rather than by documents, which is the
only reason to expect them to stay fixed.

One entry was recorded with a wrong reason and is worth naming: the snapshot
gap said `Facts` was inconsistent because "every sibling counts the policy".
That is false — `Operations`, `Targets` and `Executors` count the vocabulary
too, so `Facts` was consistent with its siblings. The real defect was narrower:
no field moved when a policy's `facts:` bounds changed or a `when:` was added.
It was fixed by adding `policy_facts` and `predicates`, not by changing what
`facts` means.

This is a living list of open items. It is not a review record: the rc.1 review
is in [post-rc-review.md](post-rc-review.md), and per-change decisions live in
`docs/change-management/`.

## Deliberately out of scope

Not gaps. Recorded so nobody reopens them as oversights:

- **`policy-inspection` and `compatibility-report` schemas.** Retired, not
  deferred. Both described the output of CLI commands that were never built, so
  each was a published promise that some tool emits this shape when none did.
  A third, `schemas/v3/agent-manifest.schema.json`, was a rival shape for the
  manifest: every document and every reference points at
  `.cervorules/agent-manifest.json`, which validates against the schema beside
  it and never matched the v3 one. They are in git history if the commands land.
  The class is now closed rather than the three instances: `validate-schemas.py`
  fails on any published schema not backed either by documents in the repository
  or by a named Go test holding its producer to it.

- **v2 `limits` in the DSL.** Not deferred: decided against. `limits.Budget` is
  `MaxTokens`, `AllowStream`, `AllowTools`, `AllowImages`, `MaxBodyBytes` —
  four of five are LLM gateway vocabulary, and `AGENTS.md` forbids adding "AI,
  provider-specific payload, or tenant concepts to core APIs". `limits/` as an
  optional leaf package is fine, because importing it is the only way to pay for
  it; the DSL is the shared surface and is not. The general case is already
  covered domain-neutrally by declaring a fact and writing a `when:` rule, so
  the consumer names the number and CervoRules never learns what a token is.
- **Generated derived facts.** Per [ADR 0014](../adr/0014-symbolic-guard-layer.md),
  `facts` owns derivation and this layer owns refutation; merging them would put
  guard evaluation on the derivation path, and a decision guard is on the hot
  path while `facts` is a bounded datalog engine with iteration budgets. A
  consumer needing derived facts derives them first and puts the results in
  `Request.Metadata`.

Both are answered in [compound-predicates.md](compound-predicates.md) §8.
