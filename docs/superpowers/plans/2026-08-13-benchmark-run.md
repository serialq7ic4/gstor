# Benchmark Run Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `gstor benchmark run` as the first disk baseline execution path and prepare release `v1.6.0`.

**Architecture:** Keep CLI parsing in `cmd/benchmark.go`; put profile definitions, dependency checks, disk eligibility, fio job generation/parsing, and orchestration in a new `common/benchmark` package. All machine-readable benchmark output stays on stdout; plans/progress/errors go to stderr.

**Tech Stack:** Go 1.24, Cobra, standard library JSON/HTTP/os/exec/syscall, existing `common/utils` helpers where appropriate.

## Global Constraints

- Existing release baseline is `v1.5.5`; this feature release target is `v1.6.0`.
- GitHub operations must use `127.0.0.1:7890` proxy.
- Required benchmark dependencies: `fio`, `lsblk`, `blkid`, `findmnt`, `fuser`, `smartctl`, root privileges.
- Profiles: `default_v1` has 25 cases and `baseline_candidate=true`; `short_v1` has 7 cases and `baseline_candidate=false`.
- `gstor benchmark run` must fail before fio starts if required dependencies or disk safety checks fail.
- `stdout` must contain final JSON only; `stderr` carries plan/progress/skip/error context.
- `--report-url` posts only completed disk results to the center API.

---

## File Structure

- Create `common/benchmark/profile.go` and `common/benchmark/profile_test.go` for profile matrices, hash calculation, media pressure mapping, and profile selection.
- Create `common/benchmark/dependencies.go` and `common/benchmark/dependencies_test.go` for root/tool readiness checks used by `gstor check` and benchmark execution.
- Create `common/benchmark/disks.go` and `common/benchmark/disks_test.go` for lsblk parsing, explicit disk resolution, and bare-disk eligibility decisions.
- Create `common/benchmark/fio.go` and `common/benchmark/fio_test.go` for fio job rendering, fio JSON parsing, and complete-profile validation.
- Create `common/benchmark/run.go` and `common/benchmark/run_test.go` for locking, serial disk execution, output file writing, and report upload orchestration.
- Create `cmd/benchmark.go` and `cmd/benchmark_test.go` for CLI flags and stdout/stderr boundaries.
- Modify `cmd/check.go` to include benchmark readiness without breaking existing controller output.
- Modify `README.md` to document `gstor benchmark run` and version target.

---

### Task 1: Profile and fio Job Model

**Files:**
- Create: `common/benchmark/profile.go`
- Create: `common/benchmark/profile_test.go`
- Create: `common/benchmark/fio.go`
- Create: `common/benchmark/fio_test.go`

**Interfaces:**
- Produces: `SelectProfile(name string) (Profile, error)`, `RenderFIOJob(profile Profile, disk DiskTarget) (string, error)`, `ValidateFIOCases(profile Profile, cases []FIOCaseResult) error`.

- [ ] Write failing tests for `default_v1` count/order/hash, `short_v1` subset count, media pressure values, sequential offset/size, and incomplete case validation.
- [ ] Run focused tests and confirm they fail because the package does not exist.
- [ ] Implement profile definitions and fio job rendering.
- [ ] Run focused tests and confirm they pass.
- [ ] Commit `feat: add benchmark profile definitions`.

### Task 2: Dependency and Bare-Disk Safety Checks

**Files:**
- Create: `common/benchmark/dependencies.go`
- Create: `common/benchmark/dependencies_test.go`
- Create: `common/benchmark/disks.go`
- Create: `common/benchmark/disks_test.go`
- Modify: `cmd/check.go`

**Interfaces:**
- Consumes: `DiskTarget` from Task 1.
- Produces: `CheckRequirements(probe Probe) RequirementReport`, `DiscoverEligibleDisks(ctx context.Context, inspector Inspector, explicit []string) ([]DiskTarget, []SkippedDisk, error)`.

- [ ] Write failing tests for missing dependency failures, non-root failure, lsblk mounted/filesystem/child exclusion, virtual disk exclusion, and explicit disk resolution.
- [ ] Run focused tests and confirm they fail.
- [ ] Implement injectable probes and conservative eligibility checks.
- [ ] Extend `gstor check` with benchmark readiness lines.
- [ ] Run focused tests and confirm they pass.
- [ ] Commit `feat: add benchmark safety checks`.

### Task 3: Benchmark Runner and CLI Wiring

**Files:**
- Create: `common/benchmark/run.go`
- Create: `common/benchmark/run_test.go`
- Create: `cmd/benchmark.go`
- Create: `cmd/benchmark_test.go`

**Interfaces:**
- Consumes: profile, dependency, disk, and fio helpers from Tasks 1-2.
- Produces: `benchmark run` command with `--profile`, `--disk`, `--output`, `--format`, and `--report-url`.

- [ ] Write failing tests for output JSON only on stdout, dependency failure before fio, report upload only for completed disks, and output file writing.
- [ ] Run focused tests and confirm they fail.
- [ ] Implement runner orchestration, `/var/lock/gstor-benchmark.lock` flock, serial per-disk fio execution, SIGINT-aware context cancellation, output file writing, and optional POST.
- [ ] Run focused tests and confirm they pass.
- [ ] Commit `feat: add benchmark run command`.

### Task 4: Documentation, Verification, and Release

**Files:**
- Modify: `README.md`
- Use existing: `check.sh`, `.github/workflows/test.yml`, `.github/workflows/release.yml`

**Interfaces:**
- Produces: release `v1.6.0` after GitHub CI and benchmark smoke validation.

- [ ] Update README command docs and JSON/report behavior notes.
- [ ] Run `gofmt`, `go test ./...`, `go vet ./...`, `golangci-lint run --timeout=5m`, and `./check.sh` with proxy where required.
- [ ] Build local binaries with `VERSION=v1.6.0 ./build.sh` and verify metadata.
- [ ] Push branch/merge to main, wait for GitHub Test workflow success through proxy.
- [ ] Test benchmark on `172.31.192.25` via the documented jump host path, using safe explicit disk selection.
- [ ] Create tag/release `v1.6.0` through the proxy and verify release workflow/assets.
