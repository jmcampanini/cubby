# cubby

`cubby` layers profile-scoped dotfiles from one or more source repositories into a host repository using safe, relative symlinks.

The v0.1 workflow is intentionally small: register sources in the host `.cubby.toml`, declare profiles in each source `cubby.toml`, then link/unlink files whose names contain selected profile suffixes.

## Install

### Homebrew `--HEAD`

```sh
brew tap jmcampanini/cubby https://github.com/jmcampanini/cubby
brew install --HEAD jmcampanini/cubby/cubby
brew upgrade --fetch-HEAD cubby
```

### Source/dev build fallback

```sh
make build
# binary at ./build/cubby
```

## Configuration

Run `cubby` from the host repo root. The host repo must contain `.cubby.toml`.

```toml
# .cubby.toml
profiles = ["work"]
# ignore_conflicts = true
# case_sensitive = true

[[source]]
name = "dotfiles"
path = "../dotfiles"
```

Each source repo must contain `cubby.toml`.

```toml
# cubby.toml
profiles = ["work", "personal"]
ignore = ["**/*.draft.*"]
```

## Profile file naming

A source file is profile-scoped when its basename contains a declared profile suffix:

- `zshrc.work`
- `nvim/init.work.lua`
- `git/config.personal.toml`

When linked, the host path preserves the same relative path and filename.

## Common workflow

```sh
cubby gitignore sync
cubby link
cubby status
cubby doctor
cubby unlink
cubby prune
```

Use `--profiles`, repeated `--profile`, or `CUBBY_PROFILES` to override host default profiles for commands that link/unlink profile files.

## Profile resolution

The raw profile list is selected from the first source available, highest to lowest:

1. `--profiles a,b,c` or repeated `--profile NAME` on the command line.
2. `CUBBY_PROFILES=a,b,c` in the environment.
3. `profiles = [...]` in `.cubby.toml`.

Raw list values are comma-split, trimmed, and deduplicated while preserving first-seen order. After raw selection, the comma-split value of the env var named by `env_profiles` (if set) is appended and deduplicated.

```toml
# .cubby.toml
profiles = ["work"]
env_profiles = "CUBBY_EXTRA"
```

Assuming a registered source declares `work`, `personal`, and `client`:

```sh
cubby profile effective                                               # -> work
CUBBY_EXTRA=personal,work cubby profile effective                     # -> work, personal
CUBBY_EXTRA=personal CUBBY_PROFILES=client cubby profile effective    # -> client, personal
CUBBY_EXTRA=personal cubby profile effective --profile client         # -> client, personal
```

Run `cubby profile effective` to see what any other command would resolve to for the current invocation; `cubby config` also includes commented effective runtime values.

## v0.1 command summary

- `cubby link [--dry-run] [--profiles LIST] [--profile PROFILE]` — create managed symlinks.
- `cubby unlink [--dry-run] [--profiles LIST] [--profile PROFILE]` — remove managed symlinks.
- `cubby status` — show managed links and drift.
- `cubby doctor` — check gitignore, sources, requested profiles, dangling links, drift, and conflicts.
- `cubby prune` — remove dangling managed symlinks.
- `cubby gitignore check` — report missing required profile ignore patterns.
- `cubby gitignore sync` — append missing required profile ignore patterns.
- `cubby profile list` — list profiles declared by sources.
- `cubby profile effective [--profiles LIST] [--profile PROFILE] [--json]` — print the effective profile list for the current invocation.
- `cubby source list` — list registered sources.
- `cubby lazygit [--source NAME]` — open `lazygit` in a registered source repo.
- `cubby config [--provenance] [--profiles LIST] [--profile PROFILE]` — print the loaded host config with effective runtime comments, optionally with provenance.
- `cubby config --validate PATH [--source-config]` — validate a host or source config file.
- `cubby docs [manual|schema|reference]` — print built-in documentation.
- `cubby completion SHELL` — generate shell completions.
- `cubby --version` — print the build version.

Data, action, and diagnostic commands support `--json` where documented. `lazygit`, `config`, `docs`, `completion`, help, and version output do not.

## Safety rules

- Cubby creates relative symlinks.
- Profile suffixes are preserved in host filenames.
- Cubby never overwrites host files.
- Conflicts fail unless `ignore_conflicts` or `--ignore-conflicts` asks Cubby to skip them.
- Cubby has no active-profile state file; profile selection comes from config, env, or flags. Run `cubby profile effective` to inspect what a given invocation resolves to.

## Known v0.1 non-goals

- No `source add/remove` commands.
- No `init` command.
- No generic `git` or `exec` command.
- No tagged release binaries yet.
- No documented `go install` install path.
