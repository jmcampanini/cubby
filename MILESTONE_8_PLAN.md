# Milestone 8 Implementation Plan

Milestone 8 is the release-readiness slice for v0.1. It should harden the
already-built command surface, make the install path match the other CLIs in
this account, and establish the black-box acceptance suite as the shipping
contract.

This plan intentionally does **not** modify `PLAN.md`. It also intentionally
replaces the earlier `go install` verification idea with a Homebrew `--HEAD`
install path.

## Outcomes

- The existing `test/e2e` package becomes the v0.1 black-box acceptance suite.
- All existing end-to-end scenarios are preserved as acceptance tests.
- Missing v0.1 acceptance coverage is added for `status`, `doctor`, `prune`,
  `lazygit`, version/build behavior, and JSON output.
- `make build` injects a git-derived version, and `cubby --version` works.
- An in-repo Homebrew `--HEAD` formula can build and test `cubby`.
- `README.md` documents supported v0.1 behavior, install instructions, safety
  rules, and known non-goals.
- Data/action/diagnostic commands support `--json` using command-specific
  envelopes.

## Non-goals

- Do not add CI in this milestone.
- Do not add a Brew-specific Make target unless a later milestone asks for it.
- Do not document `go install` as an install path.
- Do not add JSON output for `lazygit`, help, or version.
- Do not add a dedicated README JSON schema section.
- Do not add explicit ANSI/no-color acceptance tests in this milestone.

## Phase 1: Rebuild `test/e2e` into the acceptance suite

Keep the package under `test/e2e`, but split the current single large file by
behavior and centralize shared test infrastructure.

Suggested files:

- `test/e2e/harness_test.go`
- `test/e2e/link_acceptance_test.go`
- `test/e2e/gitignore_acceptance_test.go`
- `test/e2e/profile_acceptance_test.go`
- `test/e2e/source_acceptance_test.go`
- `test/e2e/config_acceptance_test.go`
- `test/e2e/status_acceptance_test.go`
- `test/e2e/lazygit_acceptance_test.go`
- `test/e2e/release_acceptance_test.go`

Harness requirements:

- Add `TestMain` that builds the real CLI once with:

  ```sh
  make build BIN=<tmp>/cubby
  ```

- Store the built binary path for all acceptance tests.
- Preserve helper behavior from the current e2e suite:
  - run commands from the host repo root via `cmd.Dir`
  - strip ambient `CUBBY_PROFILE` unless a test explicitly sets it
  - capture stdout, stderr, and exit code
  - use temp host/source repos per test
  - keep Windows symlink skips where needed
- Preserve every existing e2e scenario. Do not drop the case-collision matrix
  or other detailed edge cases.

Verification for this phase:

- `go test ./test/e2e` passes after the split.
- Existing e2e behavior is unchanged except that the binary is built once.

## Phase 2: Add versioned builds

Match the version pattern used by `cmdk` and `grove-cli`.

Implementation tasks:

- Add `cmd.Version` with default value `"n/a"`.
- Set the Cobra root command's `Version` field from `cmd.Version` so
  `cubby --version` works.
- Update `Makefile` with git-derived version injection:

  ```make
  VERSION := $(shell git describe --tags --dirty --always 2>/dev/null || date -u '+%Y-%m-%dT%H:%M:%SZ')
  LDFLAGS := -ldflags "-X github.com/jmcampanini/cubby/cmd.Version=$(VERSION)"
  ```

- Update `make build` to run `go build $(LDFLAGS) -o $(BIN) .`.
- Add acceptance coverage that the `TestMain`-built binary reports a non-empty,
  non-`n/a` version.

Verification:

- `make build` produces `build/cubby`.
- `build/cubby --version` prints a versioned `cubby` string.
- Acceptance tests prove the Makefile build path injects the version.

## Phase 3: Add Homebrew `--HEAD` formula

Follow the in-repo formula pattern from `cmdk` and `grove-cli`.

Add `Formula/cubby.rb`:

```ruby
class Cubby < Formula
  desc "Layer profile-scoped dotfiles into a host repo"
  homepage "https://github.com/jmcampanini/cubby"
  head "https://github.com/jmcampanini/cubby.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X github.com/jmcampanini/cubby/cmd.Version=HEAD-#{Utils.git_short_head}"
    system "go", "build", *std_go_args(ldflags: ldflags)
  end

  test do
    assert_match "cubby version HEAD-", shell_output("#{bin}/cubby --version")
  end
end
```

Notes:

- Do not add a Brew Make target.
- Do not add CI for Brew in this milestone.
- Manual release validation should use the self-tap pattern:

  ```sh
  brew tap jmcampanini/cubby https://github.com/jmcampanini/cubby
  brew install --HEAD jmcampanini/cubby/cubby
  brew test cubby
  ```

## Phase 4: Add missing acceptance coverage

Add black-box acceptance tests for v0.1 commands that are currently covered only
by lower-level tests or not covered end-to-end.

### `status`

Add focused acceptance coverage for:

- linked managed symlink output includes stable markers for path, source,
  profile, and target
- drift output includes stable `DRIFT` and reason markers

### `doctor`

Add focused acceptance coverage for:

- healthy setup exits zero and emits no plain output
- unhealthy setup exits non-zero and reports stable markers for:
  - missing gitignore patterns
  - missing requested profiles
  - missing/broken sources
  - dangling managed symlinks
  - conflicts

### `prune`

Add focused acceptance coverage for:

- dangling managed symlinks are removed
- valid managed symlinks are preserved
- unmanaged symlinks are preserved
- output reports removed host-relative paths

### `lazygit`

Use a fake `lazygit` executable on `PATH`; do not require the real UI tool.
Skip on Windows if script execution makes the test unreliable.

Acceptance cases:

- single registered source is selected implicitly
- explicit `--source` selects the requested source and runs fake `lazygit` from
  that source repo
- multiple registered sources without `--source` fails clearly
- unknown source fails clearly
- controlled missing-`lazygit` `PATH` fails clearly

### Release smoke

Add acceptance coverage for:

- `cubby --version` reports the Makefile-injected version
- `cubby --help` exposes the v0.1 command surface

## Phase 5: Add broad `--json` support

Add `--json` to every data/action/diagnostic command:

- `cubby profile list --json`
- `cubby source list --json`
- `cubby status --json`
- `cubby doctor --json`
- `cubby prune --json`
- `cubby gitignore check --json`
- `cubby gitignore sync --json`
- `cubby link --json [--dry-run]`
- `cubby unlink --json [--dry-run]`

Do not add `--json` to:

- `cubby lazygit`
- `cubby --help`
- `cubby --version`

### Shared JSON behavior

- Use command-specific envelopes, not a global `command`/`ok` wrapper.
- Encode JSON with `SetEscapeHTML(false)`.
- Use `/`-normalized paths in JSON.
- Omit empty optional fields with `omitempty`.
- Domain failures should emit valid JSON on stdout and then exit non-zero.
- Operational/config errors can remain plain errors on stderr for v0.1.

### Inventory JSON

`profile list`:

```json
{"profiles":["client","personal","work"]}
```

`source list`:

```json
{"sources":[{"name":"src","path":"/abs/path/to/src","profiles":["work"]}]}
```

This changes `source list --json` from the current bare array to the v0.1
envelope shape.

### Action JSON for `link` and `unlink`

Shape:

```json
{
  "dry_run": true,
  "actions": [
    {
      "kind": "create",
      "path": "nvim/init.work.lua",
      "source": "src",
      "target": "../src/nvim/init.work.lua"
    },
    {
      "kind": "conflict",
      "path": "zshrc.work",
      "source": "src",
      "reason": "host path already exists",
      "fatal": true
    }
  ]
}
```

Rules:

- `path` is the host-relative projected path.
- `source` is the registered source name.
- `target` is the relative symlink target when available.
- `reason` appears for noops, skips, conflicts, and drift-style classifications.
- `fatal` appears for fatal conflicts.
- For `link --json`, actions represent the computed plan. If fatal conflicts
  exist, the command still exits non-zero and must not mutate the filesystem.

### `status` JSON

Shape:

```json
{
  "links": [
    {
      "state": "linked",
      "path": "nvim/init.work.lua",
      "source": "src",
      "profile": "work",
      "target": "nvim/init.work.lua"
    },
    {
      "state": "drift",
      "path": "bad.work",
      "source": "src",
      "profile": "work",
      "target": "other.work",
      "reasons": ["path mismatch"]
    }
  ]
}
```

Rules:

- `state` is `linked` when there are no drift reasons and `drift` otherwise.
- `path` is host-relative.
- `target` is source-relative.
- `source` is the registered source name.
- `profile` is omitted if unknown.
- `reasons` is omitted if empty.

### `doctor` JSON

Shape:

```json
{
  "healthy": false,
  "issues": [
    {"kind":"missing_source","source":"work","message":"source \"work\" path does not exist: /tmp/work"},
    {"kind":"missing_gitignore","pattern":"*.work"},
    {"kind":"missing_profile","profile":"client"},
    {"kind":"dangling","path":"nvim/init.work.lua","source":"work","target":"nvim/init.work.lua"},
    {"kind":"drift","path":"bad.work","source":"work","target":"other.work","reasons":["path mismatch"]},
    {"kind":"conflict","path":"zshrc.work","source":"work","reason":"host path already exists"}
  ]
}
```

Rules:

- `healthy` is `len(issues) == 0`.
- Issue kinds are lowercase snake_case.
- Use structured fields where possible.
- Use `message` for source/config-ish diagnostics where the existing error text
  is the useful payload.
- Domain issues exit non-zero after JSON is written.

### `prune` JSON

Shape:

```json
{"removed":[{"path":"nvim/init.work.lua","source":"src","target":"nvim/init.work.lua"}]}
```

Rules:

- `path` is host-relative.
- `source` is the registered source name.
- `target` is source-relative.
- Empty result is `{"removed":[]}`.
- Successful prune exits zero even when nothing is removed.

### `gitignore` JSON

`gitignore check`:

```json
{"ok":false,"missing":["*.work.*","*.work"]}
```

- `ok` is `len(missing) == 0`.
- Missing patterns still exit non-zero.

`gitignore sync`:

```json
{"changed":true,"added":["*.work.*","*.work"]}
```

- `changed` is `len(added) > 0`.
- Successful sync exits zero.

## Phase 6: Add JSON acceptance coverage

Add black-box JSON assertions after the shared JSON implementation is in place.
Use Go JSON decoding in tests rather than brittle string equality except for
small inventory cases where exact order is part of the contract.

Acceptance coverage should include:

- `profile list --json` envelope
- `source list --json` envelope
- `gitignore check --json` missing-pattern domain failure with valid JSON
- `gitignore sync --json` first-run changed and second-run unchanged output
- `link --json --dry-run` with create/noop/conflict actions and non-mutation
- `link --json` fatal conflict domain failure with valid JSON and non-mutation
- `unlink --json --dry-run` with remove/skip/noop actions
- `status --json` linked and drift states
- `doctor --json` healthy and unhealthy outputs, including non-zero unhealthy
  exit with valid JSON
- `prune --json` removed entries and empty result

## Phase 7: Add `README.md`

Add `README.md` as the primary v0.1 user document.

Include:

- brief description of `cubby`
- Homebrew `--HEAD` install and upgrade instructions:

  ```sh
  brew tap jmcampanini/cubby https://github.com/jmcampanini/cubby
  brew install --HEAD jmcampanini/cubby/cubby
  brew upgrade --fetch-HEAD cubby
  ```

- source/dev build fallback:

  ```sh
  make build
  # binary at ./build/cubby
  ```

- host `.cubby.toml` example
- source `cubby.toml` example
- profile file naming rules
- common workflow:
  - `cubby gitignore sync`
  - `cubby link`
  - `cubby status`
  - `cubby doctor`
  - `cubby unlink`
  - `cubby prune`
- v0.1 command summary
- safety rules:
  - relative symlinks
  - preserve profile suffixes
  - never overwrite host files
  - conflicts fail unless skipped
  - no active-profile state file
- known v0.1 non-goals:
  - no `source add/remove`
  - no `init`
  - no generic `git`/`exec`
  - no tagged release binaries yet
  - no documented `go install` path

Mention that data/action/diagnostic commands support `--json`, but do not add a
full JSON schema section.

## Final verification

Before Milestone 8 is complete:

- `make fmt` passes.
- `make tidy` passes.
- `make test` passes.
- `make lint` passes.
- `make check` passes.
- `make build` produces `build/cubby`.
- `build/cubby --version` reports an injected version, not `n/a`.
- The rebuilt `test/e2e` acceptance suite covers the full v0.1 surface.
- Manual Homebrew validation succeeds on macOS:

  ```sh
  brew tap jmcampanini/cubby https://github.com/jmcampanini/cubby
  brew install --HEAD jmcampanini/cubby/cubby
  brew test cubby
  ```
