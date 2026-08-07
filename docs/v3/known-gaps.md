# v3 Known Gaps

Verified, unfixed defects. Each entry records how it was checked so a later
reader can confirm it still holds instead of trusting this file.

There are none open right now.

Last verified against `7e53044` on 2026-08-07, the rc.7 release commit. All
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
