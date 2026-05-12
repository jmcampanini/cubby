# Milestone 3 Plan: Profile Selection and Discovery Slice

This plan expands Milestone 3 from `PLAN.md` and records the design decisions
settled before implementation. It intentionally does not edit `PLAN.md` or
`SPEC.md`; where this file differs, it is the working plan for Milestone 3.

## Outcome

The working link/unlink flow supports realistic profile defaults, flag/env
selection, source-declared profile availability, source ignore rules, and
scriptable profile discovery output.

## Settled decisions

- Host profiles are top-level defaults, not per-source source options.

  ```toml
  profiles = ["work", "personal"]

  [[source]]
  name = "shared"
  path = "../shared"
  ```

- Source `cubby.toml` still declares which profiles that source provides.

  ```toml
  profiles = ["work", "client-acme"]
  ignore = []
  ```

- Effective profile selection precedence is:

  ```text
  defaults < host .cubby.toml < CUBBY_PROFILE < --profile
  ```

- Host `profiles` are defaults only, not an allowlist. `--profile` and
  `$CUBBY_PROFILE` may select profiles not listed in host config.
- There is no implicit "all profiles" behavior. If flags/env/host defaults
  produce no selected profiles, profile-scoped commands error.
- Every selected profile must be declared by at least one registered source.
  Validate this before link/unlink side effects.
- Per source, files are linked only for selected profiles that the source
  declares.
- `--profile` is repeatable and each value may be CSV. Whitespace is trimmed;
  duplicates are removed while preserving first-seen order.
- `$CUBBY_PROFILE` uses the same CSV parsing as `--profile`.
- If any `--profile` flag is provided, flags win completely over env. A flag
  value that parses to an empty selection errors instead of falling back to env.
- `gitignore check` and `gitignore sync` continue to use the union of all
  source-declared profiles for safety, not only currently selected profiles.
- `profile list` prints available profiles from registered sources only. It
  does not include host-only defaults with no provider.
- `profile list` output is sorted ascending, one profile per line, with no
  header/table by default.
- Source `ignore` rules are evaluated against source-relative paths using `/`
  separators.
- Ignore entries support exact path matching and glob matching, including `**`
  for recursive matches.
- Ignore patterns without `/` match basenames anywhere in the source tree.
- Invalid ignore patterns are configuration errors.

## Phase 1: Migrate host schema to global profile defaults

- Change the host config schema to add top-level profiles:

  ```go
  type HostConfig struct {
      Profiles []string     `toml:"profiles" config:"profile" help:"profile to apply; repeatable or comma-separated"`
      Sources  []HostSource `toml:"source"`
  }
  ```

- Remove `Profiles` from `HostSource`.
- Keep source config profiles unchanged:

  ```go
  type SourceConfig struct {
      Profiles []string `toml:"profiles"`
      Ignore   []string `toml:"ignore"`
  }
  ```

- Update unit and e2e fixtures from per-source host profiles to top-level host
  profiles.
- Keep strict TOML loading. Old host configs with `profiles` under `[[source]]`
  are outside the Milestone 3 target schema.
- Normalize profile slices loaded from files by trimming whitespace, dropping
  empty entries, and deduping while preserving order where order matters.

## Phase 2: Use `go-config-loader` for command profile selection

- Replace hand-rolled profile flag parsing with `go-config-loader` plus
  `pflagloader` for profile-scoped commands.
- Register `--profile` from the `HostConfig.Profiles` `config:"profile"` tag.
- For `link` and `unlink`, load the host config with loaders in this order:

  ```text
  required host file loader, environment loader, pflag loader
  ```

  using env prefix `cubby`, so `CUBBY_PROFILE` overlays host defaults and
  changed `--profile` flags overlay env.

- Rely on config-loader slice semantics for CSV parsing, repeated flags,
  whitespace trimming, and deduplication.
- Preserve source loading from each source repo's required `cubby.toml`.
- Add focused tests for:
  - host default profiles used when no flag/env is present
  - `$CUBBY_PROFILE` fallback
  - flag-over-env precedence
  - repeated flags
  - CSV flag input
  - mixed repeated + CSV input
  - empty effective selection errors
  - `--profile=` with env and host defaults present, proving a changed empty
    flag errors instead of falling back

## Phase 3: Validate effective profile selection before side effects

- After loading the project and effective host profiles, compute the union of
  profiles declared by all registered sources.
- Error before link/unlink side effects if:
  - no effective profiles are selected
  - any selected profile is not declared by at least one registered source
- For each source, link/unlink only:

  ```text
  effective selected profiles ∩ source.cubby.toml profiles
  ```

- Do not error merely because a selected profile is absent from one source, as
  long as some registered source declares it.
- Add table-driven tests that intentionally keep host defaults, effective
  command selection, and source-declared profiles different from one another,
  so accidental use of the wrong profile set fails visibly.
- Keep the existing filename grammar:
  - valid: `*.<profile>.*`
  - valid: `*.<profile>`
  - invalid/unsupported: literal `.<profile>`
- Keep ignoring lookalike files for undeclared profiles.

## Phase 4: Apply source `ignore` rules during profile-file discovery

- Extend profile-file discovery to accept the source config `ignore` list.
- Use doublestar-style glob matching; the preferred implementation dependency
  is `github.com/bmatcuk/doublestar/v4` because Go's standard glob matchers do
  not implement recursive `**` semantics.
- Evaluate ignore rules against source-relative paths normalized to `/`.
- Matching rules:
  - `nvim/init.work.lua` matches exactly that source-relative path.
  - `*.draft.*` matches any basename anywhere in the source tree.
  - `init.work.lua` matches any basename exactly named `init.work.lua`.
  - `**/*.draft.*` matches recursively across directories.
- Invalid glob patterns return an error from discovery and fail the command.
- Ignored files are silently omitted from link/unlink discovery.
- Add unit tests for:
  - exact source-relative path ignore
  - basename-only exact ignore
  - basename-only glob ignore
  - recursive `**` glob ignore
  - invalid pattern errors
  - ignored files that otherwise match a selected profile

## Phase 5: Implement and verify `cubby profile list`

- Implement `cubby profile list` as the sorted union of profiles declared by all
  registered source configs.
- Do not include host default profiles unless a source also declares them.
- Output one profile per line with no header:

  ```text
  client-acme
  personal
  work
  ```

- Add command/unit or e2e coverage that verifies sorting, dedupe, and exclusion
  of host-only defaults.

## Phase 6: Preserve gitignore safety behavior

- Ensure `cubby gitignore check` and `cubby gitignore sync` continue using the
  union of all source-declared profiles.
- Do not narrow gitignore patterns to host defaults or current command-selected
  profiles.
- Update existing gitignore tests for the new host schema.

## Phase 7: Add focused end-to-end coverage

Add separate e2e tests rather than one large scenario:

1. **Host defaults:** `cubby link` with no flag/env uses top-level host
   `profiles`.
2. **Host default unlink:** after default-profile linking, `cubby unlink` with
   no flag/env removes the matching managed links.
3. **Env fallback:** `CUBBY_PROFILE=personal cubby link` uses env-selected
   profiles when no flag is present.
4. **Flag overrides env:** `CUBBY_PROFILE=work cubby link --profile personal`
   links only `personal` files.
5. **Multi-profile selection:** repeated flags and CSV input link/unlink only
   selected profile files.
6. **No selection error:** no flag, no env, and empty/missing host profiles
   exits non-zero.
7. **Unknown profile error:** selecting a profile declared by no registered
   source exits non-zero before creating links.
8. **Ignored files:** exact-path and recursive-glob ignored files are not linked.
9. **Undeclared lookalikes:** files that look profile-scoped for undeclared
   profiles are ignored.
10. **Profile list:** available profiles are source-declared, sorted, deduped,
    and one per line.

## Deferred to Milestone 4

- `--dry-run` for `link` and `unlink`. It is valuable, but it belongs with the
  conflict and safety slice because the useful preview output should include
  creates, removals, skips, and conflicts under the final conflict model.

## Verification

- Existing Milestone 1 and 2 behaviors still pass under the new host schema.
- `link` and `unlink` accept profiles from host defaults, `$CUBBY_PROFILE`, and
  `--profile` with the expected precedence.
- `--profile` supports repeated flags, CSV values, and mixed repeated+CSV input.
- A selected profile unknown to all sources fails before filesystem changes.
- Multi-profile invocation links and unlinks only selected profile files.
- Files for undeclared profiles are ignored.
- Source ignored files are not linked.
- Invalid source ignore patterns fail clearly.
- `profile list` prints the sorted union of source-declared profiles.
- `gitignore check` and `gitignore sync` check/sync patterns for every
  source-declared profile.
- `make test` passes.
- `make lint` passes.
- `make check` passes.
