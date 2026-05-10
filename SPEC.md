# cubby — Design Spec

`cubby` is a small, stow-like CLI for layering profile-scoped files from one or
more **source** dotfiles repos into a single **host** dotfiles repo via relative
symlinks.

It is intended to be invoked from the host repo. Whatever existing tooling the
host repo uses (stow, chezmoi, plain symlinks to `$HOME`) continues to work
unchanged: `cubby` only manipulates files inside the host repo's tree.

> **Terminology note.** "Host" / "source" describe repo *roles*, not symlink
> direction. Inside the host repo, `cubby` creates symlinks whose `ln(1)`
> targets point at real files inside source repos. So a symlink's "target" (in
> `ln`/POSIX sense) is *always* a file in a source repo.

---

## 1. Concepts

- **Host repo.** The user's main dotfiles repo. `cubby` is invoked from here.
  Owns the registry of source repos and global behavioral defaults.
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
[[source]]
name             = "work"
path             = "~/Code/work-dotfiles"
profiles         = ["work"]      # required, opt-in; empty/omitted = nothing links
ignore_conflicts = false         # optional, default false
```

Notes:

- `profiles` is **strict opt-in**. If omitted or empty, `cubby` will not link
  anything from this source. There is no "all declared profiles" default.
- A profile listed here that the source repo does not declare is a `doctor`
  diagnostic, not a hard error at link time.
- `ignore_conflicts` is per-source. The CLI flag `--ignore-conflicts` (or
  similar) overrides to "skip the conflicting file, link the rest, log it" —
  never to destroy existing files.

### 4.2 Source repo: `cubby.toml` (no dot prefix)

```toml
profiles = ["work", "client-acme"]

ignore = [
  "scripts/build.test.sh",   # exact paths
  "**/*.draft.*",            # globs
]
```

The source repo's config is the source of truth for what profiles it provides
and which files within it are off-limits.

---

## 5. Activation model

`cubby` is **purely per-invocation** — there is no stored "active profiles"
state file. The set of currently active profiles is implicit: it is whatever
symlinks happen to exist in the host repo right now.

Profile selection per command:

- `--profile <name>` flag (repeatable; CSV form may also be supported)
- `$CUBBY_PROFILE` env var as fallback default
- **Flag overrides env.** If neither is set, the command errors.

Source-scoped commands (`lazygit`, etc.):

- `--source <name>` flag.
- If exactly one source is registered, it is implicit.
- Multi-source without `--source` is an error. There is **no default-source**
  concept.

---

## 6. Conflict handling

When `cubby link --profile work` walks each registered source repo and tries
to create a symlink, four cases:

| Case | Situation                                                                       | Default behavior                                  |
| ---- | ------------------------------------------------------------------------------- | ------------------------------------------------- |
| A    | Host repo already has a regular file at the projected path                       | Error                                             |
| B    | Host repo already has a symlink pointing somewhere unexpected                    | Error                                             |
| C    | Two registered sources collide on the same projected path                        | First-registered wins; second triggers Case A     |
| D    | Host repo already has the correct symlink                                        | No-op, no error                                   |

`--ignore-conflicts` (CLI) and `ignore_conflicts = true` (per-source config)
flip A/B/C from "error" to "skip and log." Existing files are never silently
destroyed.

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
| `cubby link --profile <p>`               | Create symlinks for the named profile(s) across registered sources                        |
| `cubby unlink --profile <p>`             | Remove symlinks for the named profile(s)                                                  |
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
3. Write `host/.cubby.toml` registering the source repo with
   `profiles = ["work"]`.
4. Run `cubby link --profile work` from `host/`; assert a relative symlink
   exists at `host/nvim/init.work.lua` whose target resolves to the file in
   `src/`.
5. Run `cubby gitignore check`; assert it flags missing `*.work.*` and
   `*.work` patterns.
6. Run `cubby gitignore sync`; assert the patterns appear in
   `host/.gitignore`.
7. Run `cubby unlink --profile work`; assert the symlink is gone.

Further acceptance scenarios (conflicts, multi-source, prune, doctor,
multi-profile invocations, env-var-only invocations, etc.) to be enumerated in
the implementation plan's verification step.

---

## 13. Explicitly out of scope (for now)

- Stored "active profiles" state on disk.
- Default-source / default-profile inference.
- TUI / interactive mode.
- `source add` / `source remove` / `init` commands.
- Direct `git` or `exec` pass-through.
- The bare `.<profile>` filename form.
- Tagged binary releases.
