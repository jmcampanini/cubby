# Milestone 7 Implementation Plan: Source-Scoped Lazygit

**Outcome:** `cubby lazygit` proves source-scoped command selection by launching
`lazygit` from the correct registered source repository.

This plan is intentionally narrow. It focuses on the decisions that most affect
the implementation shape and defers broader source-scoped command features until
a future milestone needs them.

## Locked decisions

### 1. Source selection

`--source` is an exact registered source-name selector.

Rules:

- Match source names exactly and case-sensitively.
- Trim surrounding whitespace from the flag value.
- If exactly one source is registered and `--source` is absent, select that
  source implicitly.
- If multiple sources are registered and `--source` is absent, fail with a clear
  ambiguity error.
- If `--source` is explicitly provided but empty after trimming, fail instead of
  falling back to implicit selection.
- If `--source` names an unknown source, fail before running anything and list
  known source names in host registration order.
- Do not add a default-source config field.
- Do not add a `$CUBBY_SOURCE` environment variable.

### 2. Project loading

Use the existing strict project loading path, likely `config.LoadProject()`.

Implications:

- `cubby lazygit` loads the host `.cubby.toml` and every registered source's
  `cubby.toml` before launching `lazygit`.
- If any registered source is invalid, missing, or has invalid config, the
  command fails before launching `lazygit`, even when `--source` names a
  different valid source.
- Partial source loading is deferred unless a future milestone needs
  `cubby lazygit --source good` to work while another registered source is
  broken.

### 3. External command execution and testing

Use a small fakeable command-runner seam instead of calling `exec.Command`
directly from the Cobra command.

Suggested shape:

```go
type externalCommand struct {
    Name string
    Args []string
    Dir  string
}

type externalCommandRunner func(externalCommand) error
```

Tests should replace the package-level runner with a fake that records the
command name, args, working directory, and call count, and can return controlled
errors.

No test should require a real `lazygit` binary or open a real TUI.

### 4. Production process behavior

The production runner should behave like the user ran `lazygit` directly from
the selected source repo:

- Command: `lazygit`
- Args: none
- Working directory: selected source repo resolved path
- Stdin: `os.Stdin`
- Stdout: `os.Stdout`
- Stderr: `os.Stderr`
- Environment: inherited from the current process

Use fully transparent stdio because `lazygit` is an interactive TUI and should
see the real terminal streams.

### 5. Launch error handling

Distinguish Cubby selection/config errors from delegated process errors:

- Missing `lazygit` binary should return a stable, actionable Cubby error, e.g.
  `lazygit not found in PATH; install lazygit or adjust PATH`.
- If `lazygit` exits non-zero, preserve the external exit code with the existing
  `cmd.ExitError` mechanism so `main` exits with that code without printing an
  extra wrapped error.
- Other process errors should be wrapped with source context, e.g.
  `run lazygit in source "work": <error>`.

## Implementation phases

### Phase 1: Add command-runner tests first

Create `cmd/lazygit_test.go` with command-level tests that use temporary real
host/source directories and real config files, but a fake lazygit runner.

Cover:

- Single registered source is selected implicitly.
- The fake runner receives:
  - name `lazygit`
  - no args
  - `Dir` equal to the selected source's resolved path
- Multiple registered sources without `--source` fail and do not call the
  runner.
- Multiple registered sources with `--source two` run in source `two`.
- Unknown source name fails clearly, lists known source names, and does not call
  the runner.
- Explicit empty `--source=` fails and does not fall back to the implicit single
  source.
- Runner missing-binary error is converted into the chosen friendly message.
- Runner non-zero exit is returned as `ExitError` with the same exit code.
- `cubby lazygit --help` shows `--source`.

### Phase 2: Replace the placeholder command

Update `cmd/lazygit.go` so `lazygitCommand()` builds a real Cobra command:

- `Use: "lazygit"`
- `Short: "Open lazygit in a source repo"`
- `--source <name>` string flag
- `RunE` flow:
  1. Load the project with `config.LoadProject()`.
  2. Select the source using the source-selection helper.
  3. Run external command `lazygit` with `Dir` set to the selected source repo.

### Phase 3: Add source-selection helper

Add a small reusable helper in `cmd`, either in `cmd/lazygit.go` or a new
`cmd/source_selection.go` file.

Suggested shape:

```go
func selectSource(project *config.Project, requested string, requestedSet bool) (config.RegisteredSource, error)
```

The helper should:

- Trim the requested flag value.
- Distinguish absent `--source` from explicitly empty `--source=`.
- Implement implicit single-source selection only when the flag is absent.
- Implement multi-source ambiguity errors.
- Implement exact source-name lookup.
- Include known source names in unknown-source errors, using host registration
  order for deterministic output.
- Return the selected source so callers can use its resolved path directly.

### Phase 4: Add the production external runner

Add the runner beside `lazygit` or in a small `cmd/external.go` file.

It should:

- Use `exec.Command("lazygit")`.
- Set `Dir` to the selected source repo.
- Set `Stdin`, `Stdout`, and `Stderr` to `os.Stdin`, `os.Stdout`, and
  `os.Stderr`.
- Rely on the default inherited environment.
- Detect missing-binary errors and return the stable friendly message.
- Convert `*exec.ExitError` into `ExitError{Code: exitCode}` when possible.

### Phase 5: Validate

Run:

```sh
make test
make lint
```

Use `make check` before marking the milestone complete.

## Verification

- A single registered source is selected implicitly.
- Multiple registered sources require `--source`.
- Unknown source names produce a clear error with known names.
- Explicit empty `--source=` errors and does not silently select anything.
- The command launches `lazygit` with the selected source repo as its working
  directory.
- Missing `lazygit` produces a stable actionable error.
- Non-zero `lazygit` exits preserve the delegated exit code.
- Tests do not require `lazygit` to be installed.

## Non-goals

- No default-source config value.
- No `$CUBBY_SOURCE` environment variable.
- No profile-selection interaction.
- No generic `cubby exec`, `cubby git`, or argument pass-through command.
- No real-TUI end-to-end test that requires `lazygit` to be installed.
- No partial project loading for broken unselected sources.
