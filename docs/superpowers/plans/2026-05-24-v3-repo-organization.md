# v3 Repo Organization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Finish the v3 repository organization pass by indexing docs, organizing scripts, and enforcing the v3 modular-root contract.

**Architecture:** Keep changes split by merge surface. Documentation indexes land first, script path moves land second, and v3 root hardening lands last after paths are stable.

**Tech Stack:** Go repoquality tests, CI, shell scripts, Markdown docs.

---

### Task 1: Documentation Topic Indexes

**Files:**
- Create: `docs/v3/README.md`
- Create: `docs/performance/README.md`
- Create: `docs/change-management/README.md`
- Create: `docs/operations/README.md`
- Create: `docs/reports/README.md`
- Modify: `docs/README.md`
- Test: `internal/repoquality/docs_topic_indexes_test.go`

- [ ] Write a repoquality test that fails when topic README files or key links are missing.
- [ ] Add the topic README files with concrete links.
- [ ] Update `docs/README.md` to link to each topic index.
- [ ] Run `go test -count=1 ./internal/repoquality`.

### Task 2: Script Domain Layout

**Files:**
- Move: `scripts/setup-go-ci.sh` -> `scripts/ci/setup-go-ci.sh`
- Move: `scripts/quality-gates.sh` -> `scripts/ci/quality-gates.sh`
- Move: `scripts/release/build-artifacts.sh` -> `scripts/release/build-artifacts.sh`
- Move: `scripts/release/check.sh` -> `scripts/release/check.sh`
- Move: `scripts/verify-generic-package.sh` -> `scripts/release/verify-generic-package.sh`
- Move: `scripts/verify-oci-tools.sh` -> `scripts/release/verify-oci-tools.sh`
- Move: `scripts/performance/report.sh` -> `scripts/performance/report.sh`
- Move: `scripts/performance/profile.sh` -> `scripts/performance/profile.sh`
- Move: `scripts/performance/benchmark-history-check.sh` -> `scripts/performance/benchmark-history-check.sh`
- Move: `scripts/sonar/scan.sh` -> `scripts/sonar/scan.sh`
- Move: `scripts/sonar/reset-local.sh` -> `scripts/sonar/reset-local.sh`
- Modify: `.github/workflows/*.yml`
- Modify: `docs/**/*.md`
- Test: repoquality script/workflow tests.

- [ ] Update repoquality tests to expect the new script paths and verify they fail.
- [ ] Move scripts with `git mv`.
- [ ] Rewrite workflow and doc references.
- [ ] Run `go test -count=1 ./internal/repoquality`.

### Task 3: v3 Modular Root Contract

**Files:**
- Modify: `internal/repoquality/v3_modular_root_contract_test.go`
- Modify: `docs/v3/module-layout.md`
- Modify: `docs/v3/modular-boundaries.md`
- Modify: `AGENTS.md`

- [ ] Add a test that parses `v3/*.go` and fails if root exports anything beyond `ModulePath`.
- [ ] Add docs explaining marker-only root and required subpackage imports.
- [ ] Run v3 and repoquality tests.

### Verification

- [ ] `go test -count=1 ./...`
- [ ] `go test -cover ./...`
- [ ] `go vet ./...`
- [ ] `go mod verify`
- [ ] `cd v3 && go test -count=1 ./...`
- [ ] `cd v3 && go vet ./...`
- [ ] `cd v3 && go mod verify`

