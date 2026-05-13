# Milestone 4 Plan: Conflict and Safety Slice

## Outcome

`cubby link` is safe to run in a real host repo because conflicts are planned,
detected, reported, and never resolved by deleting or overwriting user files.
`cubby link` and `cubby unlink` also support `--dry-run` previews.

This milestone turns the current immediate link/unlink behavior into a small
plan-then-apply flow. The plan is collected before mutation so default conflict
handling can report every conflict and avoid partial changes.

## Decisions

- Conflict-skipping CLI flag: `--ignore-conflicts`.
- Host config owns host-wide conflict policy:

  ```toml
  profiles = ["work"]
  ignore_conflicts = true

  [[source]]
  name = "src"
  path = "../src"
  ```

  `ignore_conflicts` is top-level in host `.cubby.toml`. It does **not** live
  in source `cubby.toml` and is **not** source-specific.
- `--ignore-conflicts` is global for the invocation and has the same effect as
  top-level host `ignore_conflicts = true` for that run.
- Default `link` behavior collects all planned actions and conflicts before
  mutating. If any unignored conflict exists, `link` exits non-zero and performs
  no creates.
- Conflict-skipping mode skips conflicting files, links non-conflicting files,
  reports skipped paths, and exits zero when all otherwise-fatal conflicts were
  skippable.
- Dry-run output is stable, human-readable, one action per line.
- Dry-run exit code mirrors the real command:
  - `cubby link --dry-run` exits non-zero if the real command would fail on
    conflicts.
  - `cubby link --dry-run --ignore-conflicts` exits zero for skippable conflicts.
  - config/profile/discovery errors remain non-zero.
- `unlink` treats regular files and unexpected symlinks at projected paths as
  `SKIP`, not conflicts. It only removes symlinks that resolve to the expected
  source file.
- Cross-source collisions are detected by building a host-path planning map
  across all loaded sources before mutation.
- Correct symlink detection compares resolved absolute clean target paths, not
  exact symlink text.
- Cubby links regular source files only. Source directories and source symlinks,
  including symlinks to directories, are not projected as link targets.
- Test shape: end-to-end tests first, then focused unit tests for planner/action
  classification.

## Terminology

- **Projected host path:** `hostRoot + sourceRelativePath`.
- **Correct symlink:** a host symlink whose resolved absolute clean target path
  equals the expected source file path.
- **Unexpected symlink:** a host symlink at the projected path that does not
  resolve to the expected source file path.
- **Host-path collision:** two or more planned source files project to the same
  host-relative path.
- **Fatal conflict:** a conflict that is not covered by host-wide
  `ignore_conflicts` or the global `--ignore-conflicts` flag.
- **Skippable conflict:** a conflict covered by host-wide `ignore_conflicts` or
  `--ignore-conflicts`.

## Output Contract

Dry-run and conflict-skip output should be stable enough for tests and simple
scripts. Use source-relative paths for readability and include source names when
needed to disambiguate.

Recommended line prefixes:

```text
CREATE <relpath> -> <relative-symlink-target> [source=<name>]
REMOVE <relpath> [source=<name>]
NOOP <relpath> already linked [source=<name>]
SKIP <relpath> <reason> [source=<name>]
CONFLICT <relpath> <reason> [source=<name>]
```

Rules:

- `--dry-run` prints planned `CREATE`, `REMOVE`, `NOOP`, `SKIP`, and `CONFLICT`
  records without mutating the filesystem.
- A real successful `link --ignore-conflicts` reports skipped conflicts with
  `SKIP` records.
- Default real `link` with fatal conflicts reports `CONFLICT` records and exits
  non-zero before creating anything.
- Tests may assert exact prefixes and important substrings rather than full
  prose if that keeps messages evolvable.

## Link Planning Semantics

For every source/profile file selected by the effective profile set:

1. Compute source absolute path and projected host path.
2. Add the item to a planning map keyed by host-relative path.
3. Classify any host-path collision before filesystem mutation:
   - If multiple sources project to the same host-relative path, the first
     registered source is the winner.
   - Later source items become conflicts for that path.
   - If conflict skipping applies to the later source item, it becomes `SKIP`.
   - Otherwise it is a fatal `CONFLICT`.
4. For each non-colliding planned item, inspect the host path:
   - Missing path: `CREATE`.
   - Correct symlink: `NOOP`.
   - Regular file or directory: conflict.
   - Unexpected symlink: conflict.
   - Other filesystem errors: fatal error.
5. Conflict classification:
   - If global `--ignore-conflicts` is set, conflicts become `SKIP`.
   - Else if top-level host `ignore_conflicts = true`, conflicts become `SKIP`.
   - Else conflicts remain fatal `CONFLICT` records.
6. If any fatal conflict exists in non-dry-run default mode, apply nothing and
   return non-zero.
7. Otherwise apply `CREATE` actions only. `NOOP` and `SKIP` do not mutate.

Existing regular files, directories, and unexpected symlinks are never removed,
replaced, or overwritten.

## Unlink Planning Semantics

For every selected source/profile file:

1. Compute source absolute path and projected host path.
2. Inspect the host path:
   - Missing path: `NOOP` or silent no-op.
   - Correct symlink: `REMOVE`.
   - Regular file or directory: `SKIP`.
   - Unexpected symlink: `SKIP`.
   - Other filesystem errors: fatal error.
3. `--dry-run` prints planned records and removes nothing.
4. Non-dry-run removes only `REMOVE` records.
5. Parent directories are not removed in this milestone.

`--ignore-conflicts` is accepted on `unlink` only if implementing it naturally
falls out of shared command wiring; it has no effect because unlink's projected
regular files and unexpected symlinks are skips, not conflicts.

## Implementation Phases

### Phase 1: Add end-to-end tests first

Add e2e coverage for user-visible behavior:

- Regular-file conflict:
  - Host has a regular file at a projected path.
  - `cubby link` exits non-zero.
  - The host file is unchanged.
  - No other symlinks are created when fatal conflicts exist.
- Unexpected-symlink conflict:
  - Host has a symlink at a projected path pointing somewhere else.
  - `cubby link` exits non-zero.
  - The unexpected symlink is unchanged.
- Idempotent correct symlink:
  - Existing correct symlink is a no-op.
  - `cubby link` exits zero.
- CLI conflict skipping:
  - `cubby link --ignore-conflicts` skips conflicting paths.
  - Non-conflicting selected files are linked.
  - Output reports skipped paths.
  - Exit code is zero when all conflicts are skippable.
- Host config conflict skipping:
  - Top-level `ignore_conflicts = true` skips conflicts across the whole host
    invocation.
  - There is no source-specific `ignore_conflicts` behavior.
- Cross-source collision:
  - Two registered sources contain the same selected profile-relative path.
  - Default `link` reports the collision and exits non-zero before mutation.
  - With top-level host `ignore_conflicts = true` or global
    `--ignore-conflicts`, the winner can be linked and the colliding later item
    is skipped.
- Dry-run link:
  - Reports planned creates/noops/skips/conflicts.
  - Does not create symlinks or modify existing paths.
  - Exit code mirrors the equivalent real command.
- Dry-run unlink:
  - Reports planned removes/skips/noops.
  - Does not remove symlinks.
- Unlink skip safety:
  - Regular files and unexpected symlinks at projected paths are left alone and
    reported as `SKIP` in dry-run.

### Phase 2: Add command flags

- Add `--ignore-conflicts` to `link`.
- Add `--dry-run` to `link` and `unlink`.
- Keep existing profile selection semantics unchanged.
- If `--ignore-conflicts` is shared with `unlink`, document that it is inert for
  now; otherwise only expose it on `link`.

### Phase 3: Introduce a link/unlink planner

Create or extend an internal package, likely `internal/linkops`, with planning
structures that are easy to table-test.

Suggested types:

```go
type ActionKind string

const (
    ActionCreate   ActionKind = "create"
    ActionRemove   ActionKind = "remove"
    ActionNoop     ActionKind = "noop"
    ActionSkip     ActionKind = "skip"
    ActionConflict ActionKind = "conflict"
)

type Action struct {
    Kind          ActionKind
    SourceName    string
    SourceRoot    string
    SourcePath    string
    HostPath      string
    RelPath       string
    LinkTarget    string
    Reason        string
    Fatal         bool
}
```

The planner should separate:

- discovery input: selected source files already discovered by existing profile
  discovery code;
- classification: host path state, symlink correctness, collisions, skip policy;
- rendering: stable output lines;
- applying: filesystem mutation for create/remove actions only.

### Phase 4: Implement link plan collection

- Load project and validate selected profiles before planning, as today.
- Discover selected files for all eligible sources.
- Build a host-relative-path map before mutation.
- Classify cross-source collisions deterministically by host source order.
- Classify existing host paths.
- Convert conflicts to skips when global CLI or host-wide config ignore applies.
- Return a complete plan to the command layer.

### Phase 5: Implement link apply behavior

- If plan has fatal conflicts and not dry-run:
  - render conflicts;
  - do not apply creates;
  - return a non-zero error.
- If dry-run:
  - render all relevant plan records;
  - apply nothing;
  - return non-zero only if the equivalent real command would fail.
- Otherwise:
  - create parent directories for `CREATE` actions;
  - create relative symlinks;
  - skip `NOOP` and `SKIP` actions;
  - never touch conflict paths.

### Phase 6: Implement unlink plan/apply behavior

- Reuse symlink correctness detection.
- Plan `REMOVE` only for symlinks pointing at the expected source file.
- Plan `SKIP` for regular files, directories, and unexpected symlinks.
- `--dry-run` renders without removing.
- Non-dry-run removes only `REMOVE` actions.

### Phase 7: Add focused unit tests

Add table-driven tests for planner/action classification:

- missing host path -> create;
- correct symlink -> noop;
- equivalent symlink target text resolving to same file -> noop;
- regular file -> fatal conflict by default;
- directory -> fatal conflict by default;
- unexpected symlink -> fatal conflict by default;
- global ignore converts conflicts to skips;
- host top-level `ignore_conflicts = true` converts conflicts to skips;
- source collision is fatal by default;
- source collision is skippable under global CLI or host-wide config ignore;
- fatal conflict prevents any create in default apply mode;
- unlink correct symlink -> remove;
- unlink regular file/unexpected symlink -> skip;
- dry-run apply mutates nothing.

### Phase 8: Keep existing behavior green

Verify existing Milestone 1-3 behaviors remain unchanged:

- profile selection precedence;
- source ignore rules;
- profile list output;
- gitignore check/sync union behavior;
- single-source link/unlink smoke flow;
- idempotent relinking.

## Verification

- `make test` passes.
- `make lint` passes.
- `make check` passes.
- Default conflicts exit non-zero.
- Existing regular files are never overwritten.
- Existing directories are never overwritten.
- Unexpected symlinks are never replaced.
- Correct symlinks remain idempotent no-ops.
- Default `link` with fatal conflicts performs no creates.
- Conflict-skipping mode exits zero for skippable conflicts, links
  non-conflicting files, and reports skipped paths.
- Host top-level `ignore_conflicts = true` skips conflicts across all sources.
- Global `--ignore-conflicts` skips conflicts across all sources for that run.
- Cross-source collisions are detected before mutation.
- Dry-run reports planned work/conflicts/skips and does not mutate the
  filesystem.
- Dry-run exit code mirrors the equivalent real command.
- `unlink --dry-run` reports removals without removing symlinks.

## Deferred

- Pattern-level conflict ignore rules.
- Dangerous conflict resolution modes such as adopt, backup, replace, or force.
- Empty parent directory cleanup after unlink.
- JSON dry-run output.
- Rich output styling beyond stable line-oriented records.
- Status/doctor/prune reporting of conflicts and drift, except where existing
  commands already have tests.
