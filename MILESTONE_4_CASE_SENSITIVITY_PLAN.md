# M4 Case Sensitivity Plan

## Outcome

Cubby has an explicit, host-wide case-sensitivity policy for projected host
paths. By default, Cubby behaves case-insensitively so a host repo is safe across
common macOS, Windows, and Linux workflows. Case-only path ambiguities are
planned, reported, and handled before mutation.

This plan is split into two implementation phases:

1. **Phase 1:** detect case-only collisions among planned source files before
   applying `link` actions.
2. **Phase 2:** also detect case-only conflicts between planned source files and
   existing host paths.

Implement both phases, but do them one at a time. Phase 1 is the first required
fix. Phase 2 extends the same policy to existing host filesystem state.

Scope notes:

- This plan applies to selected regular source profile files only. Cubby does
  not project source directories or source symlinks, including symlinks to
  directories.
- Host symlinks are considered only as existing projected file-path state:
  exact correct symlinks can be `NOOP`, and unexpected symlinks are conflicts or
  skips. Cubby does not traverse or manage directory symlinks as link targets.
- This milestone does not attempt full Unicode filesystem normalization or full
  Unicode case-folding semantics. The initial policy uses stable cleaned paths
  plus simple lowercase folding; this is sufficient for the ASCII-oriented
  dotfile names covered here and can be extended later if needed.

## Configuration Contract

Case behavior is host-wide and top-level in host `.cubby.toml`.

```toml
profiles = ["work"]
ignore_conflicts = false
case_sensitive = false

[[source]]
name = "src"
path = "../src"
```

Rules:

- `case_sensitive = false` is the default.
- `case_sensitive = false` means projected host paths are compared
  case-insensitively for safety.
- `case_sensitive = true` allows case-distinct projected paths when the user
  intentionally wants that behavior.
- The setting belongs to the host config, not source config.
- It is not source-specific.
- If wired through existing config-loader flags/env, expose:
  - `--case-sensitive`
  - `CUBBY_CASE_SENSITIVE=true`
- Do not add a separate `--case-insensitive` flag unless a later UX pass needs
  it.

## Output Contract

Use the existing stable action prefixes:

```text
CREATE <relpath> -> <relative-symlink-target> [source=<name>]
NOOP <relpath> already linked [source=<name>]
SKIP <relpath> <reason> [source=<name>]
CONFLICT <relpath> <reason> [source=<name>]
```

Recommended case-collision reason wording:

```text
path case collision with <winner-relpath>
```

Example default failure:

```text
CONFLICT FOO.work path case collision with foo.work [source=src]
```

Example conflict-skipping behavior:

```text
SKIP FOO.work path case collision with foo.work [source=src]
```

Tests should assert stable prefixes and important substrings, not necessarily
full prose.

Terminology:

- **Source-source case collision:** two selected source files, from the same or
  different registered sources, project to the same host path after case
  folding. Both path strings are already known from source discovery.
- **Source-host case conflict:** one selected source file projects to a path that
  collides by case with an already-existing host path. Cubby must collect the
  host path's actual spelling before comparing strings.

## Filesystem Assumptions and Testability

This feature does not require tests to run on a case-sensitive filesystem.

Most behavior is planner/config behavior and can be verified on either
case-insensitive or case-sensitive hosts:

- default `case_sensitive = false` detects case-only planned collisions;
- `--ignore-conflicts` and `ignore_conflicts = true` convert case conflicts to
  skips;
- `--dry-run` reports case conflicts/skips and mutates nothing;
- fatal case conflicts prevent partial creates;
- `case_sensitive = true` disables Cubby's case-policy collision classification.

On a case-insensitive filesystem, tests must not require both `foo.work` and
`FOO.work` to physically exist at the same time. Even when Cubby's
`case_sensitive = true` policy allows the plan, the operating system may reject
or alias the second create. That is an OS/filesystem limitation, not a Cubby
policy failure.

Therefore, e2e tests for `case_sensitive = true` should assert that Cubby does
not report a case-policy `CONFLICT`/`SKIP`. They may accept a normal filesystem
create error on case-insensitive hosts. Focused planner unit tests can fully
verify both `case_sensitive = false` and `case_sensitive = true` behavior without
filesystem dependence.

A case-sensitive filesystem can optionally add stronger e2e coverage that both
case-distinct links are created under `case_sensitive = true`, but that coverage
is not required for this milestone.

## Shared Semantics

When `case_sensitive = false`:

- Collision keys should normalize each relative path in a stable way.
- Use `filepath.Clean` plus simple lowercase folding for the planning key.
- Full Unicode case folding and filesystem Unicode normalization are explicitly
  out of scope for this milestone.
- Preserve the original relative path in rendered output.
- The first planned item wins.
- Later colliding items become either:
  - fatal `CONFLICT` records by default, or
  - `SKIP` records when `ignore_conflicts = true` or `--ignore-conflicts` is
    active.
- Fatal case conflicts prevent all `CREATE` actions.
- Dry-run reports the same plan and mirrors the real command exit code.

When `case_sensitive = true`:

- Case-only path differences do not collide during case-policy planning.
- Existing exact-path collision behavior still applies.
- The real filesystem may still reject creates on a case-insensitive host; this
  setting is an explicit user opt-in and does not guarantee the OS can represent
  both paths.

## Phase 1: Planned Source-Source Case Collisions

### Capability

Detect case-only collisions among selected source files before filesystem
mutation.

Examples that collide by default:

```text
source one: foo.work
source two: FOO.work
```

```text
same source: nvim/init.work.lua
same source: nvim/INIT.work.lua
```

### Link Planning Semantics

For every selected source/profile file:

1. Compute the source absolute path and projected host relative path.
2. Compute the collision key:
   - if `case_sensitive = true`: clean relative path;
   - if `case_sensitive = false`: clean relative path, then case-fold it.
3. Insert into the host-path planning map keyed by the collision key.
4. If the key is already present:
   - the first registered item is the winner;
   - the later item is a case/path collision;
   - default mode records fatal `CONFLICT`;
   - conflict-skipping mode records `SKIP`.
5. Continue collecting all planned records before mutation.
6. If any fatal conflict exists, apply nothing.

### User Experience

Default:

```sh
cubby link
```

- reports `CONFLICT` for later case-colliding items;
- exits non-zero;
- creates no symlinks.

Dry-run:

```sh
cubby link --dry-run
```

- reports planned creates/noops/conflicts;
- exits non-zero if the real command would fail;
- creates nothing.

Conflict-skipping:

```sh
cubby link --ignore-conflicts
```

- creates winner/non-conflicting links;
- skips later case-colliding items;
- exits zero if all conflicts are skippable.

Opt-in case-sensitive mode:

```sh
cubby link --case-sensitive
```

or:

```toml
case_sensitive = true
```

- does not treat case-only planned paths as collisions.

### Phase 1 Tests

Add focused unit tests in `internal/linkops`:

- default case-insensitive mode:
  - `foo.work` then `FOO.work` in same source -> first `CREATE`, second fatal
    `CONFLICT`;
  - same paths across two sources -> first `CREATE`, second fatal `CONFLICT`;
  - fatal case collision prevents `ApplyLink` from creating any symlink;
- conflict-skipping mode:
  - later case-colliding item becomes `SKIP`;
  - `ApplyLink` creates only winner/non-conflicting creates;
- case-sensitive mode:
  - `foo.work` and `FOO.work` do not collide in the planner;
- rendering:
  - conflict/skip output includes `path case collision` and winner relpath.

Add e2e tests:

- default `cubby link` with two selected case-only paths:
  - exits non-zero;
  - reports `CONFLICT`;
  - creates no symlinks;
- `cubby link --dry-run`:
  - reports the same conflict;
  - exits non-zero;
  - creates no symlinks;
- `cubby link --ignore-conflicts`:
  - exits zero;
  - creates winner;
  - reports `SKIP` for later case-colliding item;
  - does not fail after partial creation;
- `case_sensitive = true` or `--case-sensitive`:
  - planner does not report the case-only collision.

For the final e2e assertion, follow the filesystem assumptions above: avoid
requiring both symlinks to exist on case-insensitive CI filesystems. It is enough
to prove Cubby does not classify the pair as a policy collision; if the OS
rejects the second create, that may still be a normal filesystem error under
explicit case-sensitive mode.

## Phase 2: Existing Host Path Case Conflicts

### Capability

When `case_sensitive = false`, detect conflicts between planned projected paths
and existing host paths that differ only by case.

Example:

```text
host already has: foo.work
source projects: FOO.work
```

On a case-sensitive filesystem this could otherwise create two distinct paths,
but Cubby's default host-wide policy should treat it as ambiguous and unsafe.

### Link Planning Semantics

Use **Strategy B: a bounded host case index**. The planner should make existing
host case checks by first collecting the actual spellings of relevant host path
components, then comparing strings from the plan against that index.

For each non-colliding planned source item from Phase 1:

1. Build the set of planned host-relative paths that still need host
   classification.
2. If `case_sensitive = false`, build a bounded host case index for those paths:
   - read only host directories that are relevant to planned path prefixes;
   - store actual host-relative spellings keyed by cleaned, lowercase-folded
     relative paths;
   - include parent directory prefixes, not only leaf files;
   - sort entries before storing/reporting so output is deterministic.
3. Before exact-path classification, check whether any planned path prefix has
   an indexed host entry with the same folded key but different actual spelling:
   - host `foo.work`, planned `FOO.work` -> conflict with `foo.work`;
   - host `Nvim/`, planned `nvim/init.work.lua` -> conflict with `Nvim` even if
     `init.work.lua` does not exist under `Nvim`;
   - host `nvim/Init.work.lua`, planned `nvim/init.work.lua` -> conflict with
     `nvim/Init.work.lua`.
4. If a different-cased existing host entry is found:
   - record fatal `CONFLICT` by default;
   - record `SKIP` when conflict skipping is enabled;
   - do not create the planned symlink.
5. If no host case conflict is found, inspect the exact projected host path as
   today:
   - missing -> possible `CREATE`;
   - correct symlink -> `NOOP`;
   - regular file/directory/unexpected symlink -> conflict or skip.
6. Missing parent directories have no existing-host case conflict below that
   missing parent.

Important ordering rule: host case policy must run before treating an exact path
as a correct `NOOP`. On case-insensitive filesystems, `Lstat("FOO.work")` may
succeed when the actual directory entry is `foo.work`; Cubby still needs to
report a host path case conflict under the default policy.

Recommended reason wording:

```text
host path case conflict with <existing-relpath>
```

Example:

```text
CONFLICT FOO.work host path case conflict with foo.work [source=src]
```

### Unlink Semantics

Phase 2 is primarily a link safety feature.

For `unlink`, keep the current exact projected path behavior unless a later
implementation naturally shares host-case lookup logic. If shared, the safe
behavior is:

- never remove a path found only by case-insensitive lookup;
- report it as `SKIP` in dry-run if surfaced;
- only remove exact projected paths that are correct symlinks.

### User Experience

Default:

```sh
cubby link
```

- reports `CONFLICT` for planned paths that differ only by case from existing
  host entries;
- exits non-zero;
- creates nothing.

Conflict-skipping:

```sh
cubby link --ignore-conflicts
```

- skips the path with the host case conflict;
- links unrelated non-conflicting files;
- exits zero if all conflicts are skippable.

Dry-run:

```sh
cubby link --dry-run
```

- reports the same conflict/skip records;
- mutates nothing;
- mirrors real command exit code.

### Phase 2 Tests

Add focused unit tests in `internal/linkops`:

- default case-insensitive mode:
  - existing host regular file `foo.work`, planned `FOO.work` -> fatal
    `CONFLICT`;
  - existing host directory `foo.work`, planned `FOO.work` -> fatal
    `CONFLICT`;
  - existing unexpected symlink `foo.work`, planned `FOO.work` -> fatal
    `CONFLICT`;
  - existing parent directory `Nvim`, planned `nvim/init.work.lua` -> fatal
    `CONFLICT` even when the leaf does not exist;
  - existing `nvim/Init.work.lua`, planned `nvim/init.work.lua` -> fatal
    `CONFLICT`;
  - correct symlink with different host spelling is a host case conflict before
    it can be classified as `NOOP`;
- conflict-skipping mode:
  - host case conflict becomes `SKIP`;
  - unrelated creates still apply;
- exact projected path still uses existing classifications:
  - exact correct symlink -> `NOOP`;
  - exact regular file -> existing exact-path conflict reason;
- case-sensitive mode:
  - existing `foo.work`, planned `FOO.work` does not trigger Cubby case-policy
    conflict in the planner.

Add e2e tests:

- default link with existing host path differing only by case:
  - exits non-zero;
  - reports `CONFLICT` with `case conflict`;
  - leaves existing host path unchanged;
  - creates no unrelated symlinks;
- default link with an existing parent directory differing only by case:
  - exits non-zero;
  - reports `CONFLICT` with `case conflict`;
  - does not create a sibling directory with different casing;
- dry-run:
  - reports conflict;
  - exits non-zero;
  - mutates nothing;
- ignore-conflicts:
  - reports `SKIP`;
  - links unrelated non-conflicting files;
  - leaves existing host path unchanged.

## Implementation Notes

Suggested config addition:

```go
type HostConfig struct {
    Profiles        []string `toml:"profiles" config:"profile" help:"profile to apply; repeatable or comma-separated"`
    IgnoreConflicts bool     `toml:"ignore_conflicts" config:"ignore-conflicts" help:"skip conflicting host paths instead of failing link"`
    CaseSensitive   bool     `toml:"case_sensitive" config:"case-sensitive" help:"treat projected host paths as case-sensitive"`
    Sources         []HostSource `toml:"source"`
}
```

Suggested planner option:

```go
type PlanOptions struct {
    IgnoreConflicts bool
    CaseSensitive   bool
}
```

Suggested helper/data structure shape:

```go
func collisionKey(relPath string, caseSensitive bool) string
func caseFoldPath(path string) string // simple lowercase folding for now

type HostCaseIndex struct {
    Entries map[string][]HostCaseEntry // folded relpath -> actual spellings
}

type HostCaseEntry struct {
    RelPath string // actual host-relative spelling from os.ReadDir
    IsDir   bool
}

func BuildHostCaseIndex(hostRoot string, relPaths []string) (*HostCaseIndex, error)
func (idx *HostCaseIndex) CaseVariantPrefix(relPath string) (actualRelPath string, ok bool)
```

Implementation pattern for `BuildHostCaseIndex`:

1. Clean every planned relpath and split it into path components.
2. Start from the host root for each planned path.
3. Read only the currently relevant host directories, using a cache keyed by
   absolute directory path so repeated prefixes do not reread the same
   directory.
4. Sort directory entries before processing.
5. For the next planned component, collect host entries whose names lowercase to
   the same key.
6. Add each matching entry's actual host-relative spelling to `Entries`.
7. Continue only through matching entries that are real directories. Do not
   traverse symlinked directories; Cubby does not link or manage directories as
   targets.
8. Stop descending when no matching parent directory exists; deeper host case
   conflicts cannot exist below a missing parent.

Implementation pattern for `CaseVariantPrefix`:

1. Generate cleaned prefixes for the planned relpath, e.g.
   `nvim/init.work.lua` -> `nvim`, `nvim/init.work.lua`.
2. For each prefix, look up the lowercase folded key in `Entries`.
3. If any actual spelling differs from the planned prefix, return the first
   stable sorted actual spelling as the conflict target.
4. If all indexed spellings match exactly, return no host case conflict.

Do not implement Phase 2 as "only check the parent directory after exact
`Lstat` reports missing". That misses case-insensitive filesystems where exact
`Lstat` succeeds through an alias, and it misses parent-directory case variants
when the final leaf does not exist.

## Verification

After each layer:

- `go test ./...` passes.
- `make lint` passes.
- `make check` passes.
- Default case-insensitive planned source-source collisions are detected before
  mutation.
- Default case-insensitive source-host case conflicts are detected before exact
  `NOOP`/`CREATE` classification.
- Existing parent-directory case variants are detected, not only leaf-name
  variants.
- Dry-run exit codes mirror real link behavior.
- Conflict-skipping mode skips case conflicts and does not fail after partial
  creation.
- Existing Milestone 4 conflict behavior remains unchanged for exact-path
  conflicts.
