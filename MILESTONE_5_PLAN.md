# Milestone 5 Plan: Multi-Source Inventory Slice

**Outcome:** The core link/unlink/gitignore flow is proven across multiple
registered sources, source registration is validated strongly enough for future
source-scoped commands, and `cubby source list` provides useful human and
machine-readable inventory.

## Guiding decisions

- `source list` uses the strict project loader. It fails if any registered
  source path, source config, source profile declaration, or source name is
  invalid.
- Source names are required and must match this exact grammar:

  ```text
  ^[A-Za-z0-9_-]+$
  ```

  That means names may contain ASCII letters, digits, dashes, and underscores
  only. No spaces, dots, slashes, or other punctuation.
- Source names must be unique. Duplicate names are errors because Milestone 7
  source-scoped commands need unambiguous `--source <name>` selection.
- `source list` has two output modes:
  - Default: a Lipgloss-rendered table for humans.
  - `--json`: an unstyled JSON array for scripting.
- The `path` value in both output modes is the resolved, cleaned source root
  path Cubby actually uses after expanding `~` and resolving relative paths
  against the host root.
- The `profiles` value is the source's normalized declared profile list in
  source declaration order.
- `link` and `unlink` still treat a selected profile that no registered source
  declares as a hard error before side effects.
- A selected profile missing from an individual source is only a diagnostic when
  at least one other registered source declares it.
- Missing-profile diagnostics go to stderr for `link`, `unlink`, and their
  `--dry-run` modes. Emit one line per source/profile pair, for example:

  ```text
  source "personal" does not declare selected profile "work"; skipping
  ```

- Existing stdout for planned actions, skips, conflicts, `profile list`,
  gitignore commands, and `source list --json` remains machine-readable and is
  not mixed with diagnostics.

## Phase 1: Lock down source registration validation

- Add config tests for missing/blank source names.
- Add config tests for invalid source names, including names with spaces, dots,
  slashes, and other unsupported punctuation.
- Add config tests for duplicate source names.
- Validate source names with `regexp` using `^[A-Za-z0-9_-]+$` during project
  loading.
- Reject blank source names instead of substituting synthetic names like `#1`.
- Reject duplicate source names before returning a loaded project.
- Use the validated source name everywhere downstream.
- Keep existing strict behavior for missing paths, non-directory paths, missing
  source configs, invalid source configs, and sources that declare no profiles.

## Phase 2: Implement `cubby source list`

- Add Lipgloss as a dependency if it is not already present.
- Replace the placeholder `source list` command with a real Cobra command.
- Add a `--json` bool flag.
- Load the project with `config.LoadProject()`.
- Build an inventory view model from `project.Sources`:

  ```go
  type SourceListItem struct {
      Name     string   `json:"name"`
      Path     string   `json:"path"`
      Profiles []string `json:"profiles"`
  }
  ```

- Default output:
  - Render a Lipgloss table to stdout.
  - Include columns: `NAME`, `PATH`, `PROFILES`.
  - Keep rows in host registration order.
  - Render profiles as comma-separated text with no spaces, e.g.
    `work,personal`.
- JSON output:
  - Print only JSON to stdout; do not apply Lipgloss styling.
  - Print a JSON array in host registration order, followed by a newline:

    ```json
    [{"name":"one","path":"/resolved/src1","profiles":["work","personal"]},{"name":"two","path":"/resolved/src2","profiles":["client"]}]
    ```

- Add focused command tests for:
  - `source list --json` exact JSON content.
  - default `source list` includes the expected table headers and values.
  - output order follows host registration order.
  - paths are resolved paths.
  - profiles are per-source declarations, not the host default profile list.
- Add a strict-loading test proving `source list` fails when any registered
  source is invalid.

## Phase 3: Add per-source missing-profile diagnostics

- Add a small command helper that compares the effective selected profiles to
  each source's declared profiles.
- Call the helper from `linkProfilesWithOptions` and `unlinkProfilesWithOptions`
  after global selected-profile validation succeeds and before discovery or
  filesystem mutation.
- Write diagnostics to `cmd.ErrOrStderr()`.
- Emit diagnostics in both real and `--dry-run` runs because both use the same
  selected source/profile plan.
- Do not convert these diagnostics into errors unless a selected profile is
  declared by no registered source, which is already a global validation error.
- Keep `sourceSelectedProfiles` as the implementation point that filters work to
  each source's declared-profile intersection.

## Phase 4: Prove multi-source link/unlink behavior end to end

- Add an e2e test with two registered sources and top-level host defaults that
  select profiles provided by different sources.
- Seed each source with distinct profile files and assert `link` creates
  relative symlinks to the correct source repo.
- Assert stderr includes diagnostics for selected profiles that are absent from
  individual sources, without failing the command.
- Run `unlink` with the same default selection and assert only the matching
  managed symlinks are removed.
- Add an e2e test for explicit `--profile` selection with multiple sources and
  empty host defaults.
- Add or retain e2e coverage for cross-source path collisions:
  - default mode exits non-zero without creating links;
  - conflict-skipping mode links the first-registered source's non-conflicting
    winner and reports the skipped collision.
- Add a `--dry-run` diagnostic assertion proving missing-profile diagnostics are
  still written to stderr while stdout contains only planned actions.

## Phase 5: Prove inventory and gitignore union behavior

- Add an e2e test for `source list --json` with at least two sources, asserting
  exact JSON stdout using the test's actual resolved paths.
- Add an e2e smoke assertion for default `source list` proving the table renders
  source names, resolved paths, and profiles.
- Keep or extend `profile list` tests to assert the sorted union of profiles
  declared by every source, not host defaults.
- Add an e2e test for `gitignore check` and `gitignore sync` with profiles
  spread across multiple sources.
- Assert `gitignore check` reports missing patterns for the full source-declared
  union before sync.
- Assert `gitignore sync` appends every missing union pattern once and a second
  `gitignore check` succeeds.
- Add an e2e test proving multiple registered sources with empty top-level host
  `profiles` still require an explicit `--profile` flag or `$CUBBY_PROFILE` for
  `link`/`unlink`.

## Verification

- `source list` prints a Lipgloss table with name, resolved path, and profiles
  columns.
- `source list --json` prints an unstyled JSON array of objects with `name`,
  `path`, and `profiles` fields.
- Blank source names are rejected.
- Source names containing spaces or characters outside `[A-Za-z0-9_-]` are
  rejected.
- Duplicate source names are rejected.
- `profile list` prints the sorted union of profiles declared by all registered
  sources.
- `gitignore check` and `gitignore sync` use the union of profiles declared by
  all registered sources.
- Empty top-level host profiles require explicit flag/env profile selection for
  profile-scoped commands.
- Multi-source `link` creates the expected relative symlinks from every eligible
  source.
- Multi-source `unlink` removes only matching managed links.
- Cross-source collisions remain safe: default mode fails without mutation and
  conflict-skipping mode skips/reports conflicts without overwriting user files.
- Profiles missing from an individual source are reported on stderr for `link`,
  `unlink`, and `--dry-run`, but do not block work when another source provides
  that profile.
- Existing Milestone 1-4 tests still pass.
- `make test` passes.
- `make lint` passes.
- `make check` passes.
