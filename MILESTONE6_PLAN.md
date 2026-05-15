# Milestone 6 Plan: Status, Doctor, and Prune Slice

This expands the Milestone 6 entry in `PLAN.md` into an implementation-ready
plan. It should not require an active-profile state file: current Cubby state is
whatever managed symlinks already exist in the host repo.

## Outcome

`cubby` can explain and clean the state it created:

- `cubby status` reports managed links, source/profile ownership, and drift.
- `cubby prune` removes dangling managed symlinks and leaves everything else
  alone.
- `cubby doctor` runs aggregate health checks and exits non-zero for unhealthy
  setups.

## Constraints and Non-Goals

- Do not add an active-profile state file.
- Do not change existing `link`, `unlink`, `profile list`, `source list`, or
  `gitignore` behavior except where shared helpers are refactored safely.
- `status`, `doctor`, and `prune` should stay scriptable and line-oriented.
- `prune` only removes symlinks classified as Cubby-managed and dangling.
  It does not remove parent directories.
- `lazygit` and release/distribution work remain Milestones 7 and 8.

## Working Definitions

- **Managed symlink:** a symlink inside the host repo whose target path resolves
  inside a registered source repo. For dangling targets, use the cleaned absolute
  target path lexically so stale links can still be recognized.
- **Source ownership:** the registered source whose root contains the symlink
  target. If roots overlap, prefer the most-specific root; break exact ties by
  host config order.
- **Profile ownership:** the profile matched by the target basename using the
  existing profile-file grammar and the owning source's declared profiles.
- **Dangling managed symlink:** a managed symlink whose target path no longer
  exists.
- **Drift:** a managed symlink that no longer matches what Cubby would create,
  such as a path mismatch, dangling target, ignored target, or target basename
  that no longer matches a declared profile.
- **Requested profiles for doctor:** top-level host `profiles` from
  `.cubby.toml`. `doctor` should not invent an implicit all-profiles selection.

## Phase 1: Add Tests First

- Replace the current not-implemented tests for `status`, `doctor`, and `prune`
  with behavior tests.
- Add focused unit tests for managed symlink discovery:
  - relative symlink target into a source is managed;
  - absolute symlink target into a source is managed;
  - symlink outside all registered sources is unmanaged;
  - dangling symlink into a source is managed and dangling;
  - host-relative path drift is detected;
  - source profile ownership is inferred from declared profiles;
  - source ignore rules mark an existing managed link as drift.
- Add end-to-end tests for:
  - `status` reporting linked files with source/profile information;
  - `status` reporting drift/dangling links;
  - `prune` removing dangling managed symlinks only;
  - `doctor` exiting zero for a healthy setup;
  - `doctor` exiting non-zero for missing gitignore patterns, conflicts,
    dangling links, missing sources, and missing requested profiles.

## Phase 2: Add Diagnostic-Friendly Project Loading

Current strict project loading is correct for existing commands, but `doctor`
needs to report multiple setup problems instead of failing on the first missing
source.

- Keep `config.LoadProject()` strict and unchanged for existing commands.
- Add a diagnostic loader, likely in `internal/config`, that:
  - loads the host `.cubby.toml` from the current host root;
  - resolves every registered source path;
  - returns valid `RegisteredSource` entries where possible;
  - returns structured source issues for missing paths, non-directories,
    missing `cubby.toml`, invalid source configs, and sources with no profiles;
  - preserves host config order for deterministic diagnostics.
- Reuse existing path resolution and normalization helpers so strict and
  diagnostic loading do not drift.

## Phase 3: Implement Managed Symlink Discovery

Add a small internal package, for example `internal/hostlinks`, to walk the host
repo and classify symlinks.

Suggested data shape:

```go
type ManagedLink struct {
    HostPath        string
    HostRelPath     string
    RawTarget       string
    TargetPath      string
    SourceName      string
    SourceRoot      string
    SourceRelPath   string
    Profile         string
    TargetExists    bool
    DriftReasons    []string
}
```

Discovery rules:

- Walk the host repo with `filepath.WalkDir`.
- Skip `.git` directories.
- If a registered source root is inside the host repo, skip walking inside that
  source root so source-internal symlinks are not mistaken for host state.
- Consider symlinks only; ignore regular files and directories.
- Resolve relative symlink targets from the symlink's parent directory.
- Use `filepath.EvalSymlinks` when possible for existing targets, but still use
  the cleaned absolute target path for dangling targets.
- Match targets to registered source roots with a safe inside-root check based
  on `filepath.Rel`.
- Sort results by host-relative path, then source name for stable output.

Drift rules:

- Target missing: `dangling`.
- Host relative path differs from target source-relative path: `path mismatch`.
- Target basename no longer matches any declared profile for that source:
  `unknown profile`.
- Target is excluded by the source's current `ignore` rules: `ignored`.

## Phase 4: Implement `cubby status`

- Load the strict project config.
- Discover managed symlinks in the host repo.
- Print stable, line-oriented output that includes:
  - host-relative path;
  - owning source name;
  - inferred profile, when known;
  - source-relative target path;
  - drift/dangling reason, when present.
- Exit zero for successful reporting, even when drift is present. `doctor` is
  responsible for health-check exit codes.

## Phase 5: Implement `cubby prune`

- Load the strict project config.
- Discover managed symlinks in the host repo.
- Remove only links classified as managed and dangling.
- Print each removed host-relative path.
- Leave valid managed links, unmanaged symlinks, unexpected regular files, and
  directories untouched.
- If source roots/configs cannot be loaded, fail before mutation; `doctor`
  reports missing-source health issues separately.

## Phase 6: Implement `cubby doctor`

Use diagnostic-friendly loading so one bad source does not hide other health
issues.

Checks:

- Missing or invalid registered sources.
- Missing required `.gitignore` patterns for the union of profiles declared by
  valid sources.
- Missing requested profiles: host top-level `profiles` that no valid source
  declares.
- Dangling managed symlinks.
- Managed-link drift.
- Link conflicts for host top-level requested profiles, when that selection is
  non-empty and declared by at least one valid source. Reuse profile discovery
  and `linkops.PlanLink` without mutating the filesystem.

Exit behavior:

- Exit zero when no issues are found.
- Exit non-zero when any issue is found.
- Prefer stable diagnostic prefixes such as `MISSING_SOURCE`,
  `MISSING_GITIGNORE`, `MISSING_PROFILE`, `DANGLING`, `DRIFT`, and `CONFLICT`
  so tests and scripts can assert behavior without depending on styling.

## Phase 7: Validation

Run the normal project checks:

- `make fmt`
- `make test`
- `make lint`
- `make check`

Milestone 6 is complete when:

- `status` reports linked files with source and profile information.
- `status` identifies drift between host symlinks and source files.
- `doctor` exits zero for a healthy setup.
- `doctor` exits non-zero and reports unhealthy setup issues.
- `prune` removes dangling managed symlinks.
- `prune` leaves valid symlinks and unmanaged symlinks untouched.
