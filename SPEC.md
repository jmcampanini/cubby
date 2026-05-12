# cubby — Design Spec

`cubby` is a small, stow-like CLI for layering profile-scoped files from one or
more **source** dotfiles repos into a single **host** dotfiles repo via relative
symlinks.

It is intended to be invoked from the host repo root. Whatever existing tooling
the host repo uses (stow, chezmoi, plain symlinks to `$HOME`) continues to work
unchanged: `cubby` only manipulates files inside the host repo's tree.

> **Terminology note.** "Host" / "source" describe repo *roles*, not symlink
> direction. Inside the host repo, `cubby` creates symlinks whose `ln(1)`
> targets point at real files inside source repos. So a symlink's "target" (in
> `ln`/POSIX sense) is *always* a file in a source repo.

---

## 1. Concepts

- **Host repo.** The user's main dotfiles repo. `cubby` is invoked from the
  host repo root, using Stow-like ergonomics rather than walking upward to
  discover a root. Owns the registry of source repos and global behavioral
  defaults.
- **Source repo.** A separate git repo containing profile-scoped files.
  Self-describing: declares which profiles it provides and which files to
  ignore.
- **Profile.** A logical grouping of files identified by a name (e.g., `work`,
  `client-acme`). A profile can span multiple source repos. Many profiles can
  be active in the host at the same time.
- **Profile file.** A file in a source repo whose name matches `*.<profile>.*`
  or `*.<profile>` for a profile that the source repo declares. Anything else
  is ignored.

---

## 2. File naming grammar

Two patterns are valid for profile files:

| Pattern             | Examples                                              |
| ------------------- | ----------------------------------------------------- |
| `*.<profile>.*`     | `nvim/init.work.lua`, `archive.work.tar.gz`           |
| `*.<profile>`       | `Makefile.work`, `.gitignore.work`, `bin/deploy.work` |

The literal `.<profile>` form (a hidden file whose entire name is the profile
suffix, e.g., `.work`) is **not** supported.

Profile names are recognized by registration, not heuristic. `cubby` only
treats a dot-segment as a profile if the source repo's `cubby.toml` declares
that name in `profiles`. This avoids false positives like `script.test.sh`
(where `test` is not a profile).

The source repo can additionally provide an `ignore` list to exclude files
that match the pattern but should not be projected.

---

## 3. Path mapping & symlink behavior

- The source repo's tree is mirrored into the host repo at the **same relative
  path**. `source-repo/nvim/init.work.lua` projects to
  `host-repo/nvim/init.work.lua`.
- The **profile suffix is preserved** in the projected filename. Downstream
  tooling (the host repo's stow/chezmoi setup, your shell rc, etc.) is
  responsible for whatever it does with `*.work.*` files.
- Symlinks are **relative** (matching `stow`'s default), computed via Go's
  `filepath.Rel`. The real file lives in the source repo; the symlink lives in
  the host repo.
- **Idempotent**: re-running `cubby link` over a correctly-linked file is a
  no-op.

---

## 4. Configs

Both configs are TOML, loaded via `github.com/jmcampanini/go-config-loader`
(precedence: flags > env > files > defaults).

### 4.1 Host repo: `.cubby.toml` (dot-prefixed)

```toml
profiles = ["work", "personal"] # optional command defaults for link/unlink
ignore_conflicts = false         # optional global conflict-skip default

[[source]]
name = "work"
path = "~/Code/work-dotfiles"
```

Notes:

- Top-level `profiles` are host defaults for profile-scoped commands. If no
  `--profile` flag or `$CUBBY_PROFILE` env var is provided, `link` and `unlink`
  use this list.
- Host `profiles` are **not** an allowlist. Flags and env may select profiles
  not listed in the host defaults, but every selected profile must be declared
  by at least one registered source before any link/unlink side effects occur.
- If flags, env, and host defaults produce no selected profiles, the command
  errors. There is no implicit "all declared profiles" default.
- `profiles` do not live under `[[source]]` in the host config. Source-specific
  profile availability is declared by each source repo's `cubby.toml`.
- Top-level `ignore_conflicts` is a host-wide default for conflict skipping.
  The CLI flag `--ignore-conflicts` overrides to "skip the conflicting file,
  link the rest, log it" — never to destroy existing files.

### 4.2 Source repo: `cubby.toml` (no dot prefix)

```toml
profiles = ["work", "client-acme"]

ignore = [
  "scripts/build.test.sh",   # exact paths
  "**/*.draft.*",            # globs
]
```

The source repo's config is the source of truth for what profiles it provides
and which files within it are off-limits. Ignore entries are matched against
source-relative paths normalized to `/`; entries without `/` match basenames
anywhere in the source tree, and glob entries use doublestar-style `**`
recursive semantics.

---

## 5. Activation model

`cubby` is **purely per-invocation** — there is no stored "active profiles"
state file. The set of currently active profiles is implicit: it is whatever
symlinks happen to exist in the host repo right now.

Profile selection for profile-scoped commands uses this precedence:

```text
defaults < host .cubby.toml profiles < CUBBY_PROFILE < --profile
```

- `--profile <name>` is repeatable; each value may also be comma-separated.
- `$CUBBY_PROFILE` uses the same comma-separated parsing as `--profile`.
- **Flag overrides env.** If any `--profile` flag is provided, env and host
  defaults are ignored for that command.
- If the effective selection is empty, the command errors. There is no implicit
  "all profiles" behavior.
- If any selected profile is declared by no registered source, the command
  errors before creating or removing links.
- Per source, link/unlink applies only the intersection of the effective
  selection and that source's declared profiles.

Source-scoped commands (`lazygit`, etc.):

- `--source <name>` flag.
- If exactly one source is registered, it is implicit.
- Multi-source without `--source` is an error. There is **no default-source**
  concept.

---

## 6. Conflict handling

When `cubby link` applies a selected/default profile such as `work` and tries
to create a symlink, four cases:

| Case | Situation                                                                       | Default behavior                                  |
| ---- | ------------------------------------------------------------------------------- | ------------------------------------------------- |
| A    | Host repo already has a regular file at the projected path                       | Error                                             |
| B    | Host repo already has a symlink pointing somewhere unexpected                    | Error                                             |
| C    | Two registered sources collide on the same projected path                        | First-registered wins; second triggers Case A     |
| D    | Host repo already has the correct symlink                                        | No-op, no error                                   |

`--ignore-conflicts` (CLI) and top-level `ignore_conflicts = true` in the host
config flip A/B/C from "error" to "skip and log." Existing files are never
silently destroyed.

`--dry-run` for `link` and `unlink` previews planned creates/removals, skips,
and conflicts without mutating the filesystem. It belongs with the conflict and
safety behavior rather than profile discovery.

---

## 7. Gitignore guard

A primary concern: the public host repo must not accidentally commit
profile-scoped files.

- The required `.gitignore` patterns are **globs per profile**:
  `*.<profile>.*` and `*.<profile>` for every profile declared by every
  registered source repo.
- `cubby gitignore check` reads the union of all declared profiles, compares
  against the host's `.gitignore`, and reports missing patterns. Exits
  non-zero if anything is missing.
- `cubby gitignore sync` appends missing patterns to the host's `.gitignore`.

---

## 8. Command surface (v0.1)

| Command                                  | Purpose                                                                                  |
| ---------------------------------------- | ---------------------------------------------------------------------------------------- |
| `cubby link [--profile <p>] [--ignore-conflicts] [--dry-run]` | Create symlinks for selected/default profile(s) across registered sources |
| `cubby unlink [--profile <p>] [--dry-run]` | Remove symlinks for selected/default profile(s)                                          |
| `cubby prune`                            | Remove dangling symlinks (target file no longer exists)                                   |
| `cubby status`                           | Report what is linked, from which source, for which profile, and any drift                |
| `cubby doctor`                           | Aggregate health checks (gitignore, conflicts, dangling, missing sources, etc.)           |
| `cubby profile list`                     | Print the union of profiles declared across all registered sources                        |
| `cubby source list`                      | Print registered source repos                                                             |
| `cubby gitignore check`                  | Verify required `.gitignore` patterns are present                                         |
| `cubby gitignore sync`                   | Add missing `.gitignore` patterns                                                         |
| `cubby lazygit [--source <name>]`        | Open lazygit inside the named (or implicit) source repo                                   |

`unlink` and `status` discover state by **walking the host repo for symlinks
whose targets resolve into a registered source repo**, then filtering by
profile. There is no on-disk state file to maintain.

There is no `source add` / `source remove` — registration is by hand-editing
`.cubby.toml`.

There is no `init` / `git` / `exec` command in v0.1.

---

## 9. Output style

- **Lipgloss only** for terminal styling (colors, bold, simple tables). No
  Bubble Tea, no full-screen TUI. The CLI must remain scriptable and
  pipe-friendly.

---

## 10. Distribution

- Homebrew tap: `jmcampanini/tap/cubby`.
- **`--HEAD`-only** formula initially (i.e., installs by building the latest
  `main`). No tagged release binaries until the surface stabilizes.
- `go install` should also work as a fallback for non-Homebrew users.

---

## 11. Source layout

- `~/Code/github.com/jmcampanini/cubby/main/` — main worktree.
- Conventional Go layout: `cmd/cubby/`, `internal/`, etc. Specifics deferred to
  the implementation plan.

---

## 12. Acceptance / verification sketch

End-to-end smoke test the implementation must pass:

1. Create two throwaway git repos in `/tmp/` — `host/` and `src/`.
2. Seed `src/nvim/init.work.lua` and `src/cubby.toml` declaring
   `profiles = ["work"]`.
3. Write `host/.cubby.toml` with top-level `profiles = ["work"]` and one
   `[[source]]` registering the source repo.
4. Run `cubby link` from `host/`; assert a relative symlink exists at
   `host/nvim/init.work.lua` whose target resolves to the file in `src/`.
5. Run `cubby gitignore check`; assert it flags missing `*.work.*` and
   `*.work` patterns.
6. Run `cubby gitignore sync`; assert the patterns appear in
   `host/.gitignore`.
7. Run `cubby unlink`; assert the symlink is gone.

Further acceptance scenarios (conflicts, multi-source, prune, doctor,
multi-profile invocations, env-var-only invocations, etc.) to be enumerated in
the implementation plan's verification step.

---

## 13. Explicitly out of scope (for now)

- Stored "active profiles" state on disk.
- Default-source inference and implicit all-profiles selection.
- TUI / interactive mode.
- `source add` / `source remove` / `init` commands.
- Direct `git` or `exec` pass-through.
- The bare `.<profile>` filename form.
- Tagged binary releases.
