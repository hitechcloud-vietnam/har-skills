# Package Path Consistency and Coverage Verification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development`
> Steps use checkbox (`- [ ]`) syntax.

**Goal:** Verify that Go package paths throughout the repository match the GitHub repository `hitechcloud-vietnam/har-skills`, recheck whether unit test coverage remains at 100%, and document the delivery acceptance results.

**Architecture:** Both requirements are verification tasks, not development tasks: no new features are needed. Systematically inspect repository-wide `package` declarations, `import` paths, the `go.mod` module declaration, and GoReleaser ldflags injection paths; rerun coverage tests, including race mode, to confirm stable 100% coverage. Trace the path from the go.mod module declaration through package declarations and imports, the GoReleaser ldflags `-X` paths, and the version output of the built artifact, confirming that all use the same module path: `github.com/hitechcloud-vietnam/har-skills`.

**Tech Stack:** Go 1.24（go.mod），GoReleaser v2 schema，goreleaser-action `~> v2`，cobra/viper CLI，golangci-lint，GitHub Actions

**Risks:**
- The project previously used the module path `hitechcloud-vietnam/go-har`; stale imports may remain after the rename. Mitigation: search the entire repository for the old path.
- If the ldflags injection path does not match the actual package path, version information may be silently omitted. Mitigation: simulate ldflags in a local build and verify the `--version` output.
- Test data timestamps may be regenerated during tests, leaving the worktree dirty. Mitigation: restore any such changes after verification.

---

### Task 1: Verify Repository Package Paths Match the GitHub Repository

**Depends on:** None
**Files:** Verification only; no files are modified.
- Read: `go.mod`
- Read: `cmd/har/main.go`、`cmd/har/cmd/root.go`、`cmd/har/internal/*.go`
- Read: `.goreleaser.yaml`
- Read: `CLAUDE.md`

- [x] **Step 1: Verify the go.mod module declaration matches the GitHub remote path**

Run: `head -1 go.mod && git remote -v`
Expected:
  - Output contains: `module github.com/hitechcloud-vietnam/har-skills`
  - Output contains: `git@github.com:hitechcloud-vietnam/har-skills.git`
  - The owner/name components match exactly.

- [x] **Step 2: Verify cmd/har subpackage declarations and import paths**

Run: `find cmd -name "*.go" | while read f; do echo "$f: $(grep -m1 '^package ' $f)"; done`
Expected:
  - `cmd/har/main.go: package main`，import `github.com/hitechcloud-vietnam/har-skills/cmd/har/cmd`
  - `cmd/har/cmd/*.go: package cmd`
  - `cmd/har/internal/*.go: package internal`
  - All imports use `github.com/hitechcloud-vietnam/har-skills` as their root.

- [x] **Step 3: Search the entire repository for remnants of the old module path `hitechcloud-vietnam/go-har`**

Run: `grep -rn "hitechcloud-vietnam/go-har" --include="*.go" --include="*.yaml" --include="*.yml" --include="*.md" .`
Expected:
  - No output (no remnants found).

- [x] **Step 4: Verify the GoReleaser ldflags injection path matches the actual package path**

Run: `grep -n "X github.com" .goreleaser.yaml`
Expected:
  - `-X github.com/hitechcloud-vietnam/har-skills/cmd/har/cmd.version={{.Version}}`
  - `-X github.com/hitechcloud-vietnam/har-skills/cmd/har/cmd.commit={{.Commit}}`
  - `-X github.com/hitechcloud-vietnam/har-skills/cmd/har/cmd.date={{.Date}}`
  - It matches the package path of the `var version/commit/date` declarations in `cmd/har/cmd/root.go`.

- [x] **Step 5: Simulate ldflags in a local build to verify version injection**

Run: `go build -ldflags "-X github.com/hitechcloud-vietnam/har-skills/cmd/har/cmd.version=v0.1.2-test -X github.com/hitechcloud-vietnam/har-skills/cmd/har/cmd.commit=abc123 -X github.com/hitechcloud-vietnam/har-skills/cmd/har/cmd.date=2026-07-21" -o /tmp/har-ldflag-test ./cmd/har/ && /tmp/har-ldflag-test --version`
Expected:
  - Exit code: 0
  - Output contains: `HAR Skills v0.1.2-test`、`commit: abc123`、`date: 2026-07-21`
  - All three variables are injected successfully. Note: quote the entire ldflags string, or the shell will split the `-X` arguments.

---

### Task 2: Recheck 100% Unit Test Coverage

**Depends on:** None
**Files:** Verification only; no files are modified.
- Read: All root-package `*_cov_test.go` files and `coverage_final_test.go`.
- [x] **Step 1: Coverage in standard mode**

Run: `go test -coverprofile=/tmp/c.out -count=1 . 2>&1 | tail -1`
Expected:
  - Output contains: `coverage: 100.0% of statements`

- [x] **Step 2: Verify coverage stability in race mode**

Run: `go test . -race -cover -count=1 -timeout 120s 2>&1 | tail -1`
Expected:
  - Exit code: 0
  - Output contains: `coverage: 100.0% of statements`
  - No race warnings.

- [x] **Step 3: Confirm every function has 100% coverage**

Run: `go tool cover -func=/tmp/c.out | awk '$3 != "100.0%" && $1 != "total"'`
Expected:
  - No output (all functions have 100% coverage).

- [x] **Step 4: Restore testdata timestamp changes caused by test side effects**

Run: `git checkout testdata/full.har testdata/invalid_date.har testdata/large.har`
Expected:
  - Exit code: 0
  - `git status --short` shows no remaining testdata changes.

---

### Task 3: Document the Delivery Acceptance Results

**Depends on:** Task 1, Task 2
**Files:**
- Create: This plan document (verification evidence is recorded here).
- [x] **Step 1: Summarize the verification results**

Verification results (2026-07-21):

| Requirement | Status | Evidence |
|------|------|------|
| Package path matches the GitHub repository | ✅ Verified | `module github.com/hitechcloud-vietnam/har-skills` matches GitHub `hitechcloud-vietnam/har-skills`; no old `go-har` paths remain; ldflags match the actual package path; all three version variables were injected in a local build and verified with `--version`. |
| Unit test coverage is 100% | ✅ Verified | Standard mode: `coverage: 100.0%`; race mode: `coverage: 100.0%`; every function has 100% coverage. |

- [x] **Step 2: Commit the plan document**

Run: `git add docs/superpowers/plans/2026-07-21-package-path-and-coverage-verification.md && git commit -m "docs(plans): verify package path consistency and 100% coverage"`
