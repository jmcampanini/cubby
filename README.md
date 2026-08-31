# cubby

cubby layers profile-scoped dotfiles from one or more source repositories into a host repository as relative symlinks. A host repository registers its sources in `.cubby.toml`, each source declares its profiles in `cubby.toml`, and a file whose name carries a profile suffix (`zshrc.work`, `nvim/init.work.lua`) is linked into the host at the same relative path whenever that profile is selected. cubby never replaces an existing host file and never accesses the network.

Command help is the canonical reference: `cubby --help` and each command's `--help` describe every user-facing contract, `cubby config --help` describes configuration precedence and the file schema, and `cubby help exit-codes` describes exit statuses.

## Install

cubby distributes from HEAD only; there is no release channel or tagged binary.

### Homebrew

```sh
brew tap jmcampanini/cubby https://github.com/jmcampanini/cubby
brew install --HEAD jmcampanini/cubby/cubby
```

Upgrade to the latest commit:

```sh
brew upgrade --fetch-HEAD cubby
```

### From source

```sh
make build
# then copy ./build/cubby to a directory on your PATH
```

## Representative commands

| Command | Result |
|---|---|
| `cubby gitignore sync` | Append the `.gitignore` patterns the host needs for `.cubby.toml` and profile-scoped files. |
| `cubby link [--dry-run]` | Create symlinks for the selected profiles; `--dry-run` prints the plan instead. |
| `cubby status [--json]` | List managed links and any drift. |
| `cubby doctor [--json]` | Report health issues and exit 1 when there are any. |
| `cubby unlink [--dry-run]` | Remove the links `link` created for the selected profiles. |
| `cubby prune [--json]` | Remove managed links whose target no longer exists. |
| `cubby profile effective` | Print the profiles this invocation would use. |
| `cubby source list [--json]` | List registered sources. |
| `cubby lazygit [--source NAME]` | Open lazygit in a source repository. |
| `cubby config [--provenance]` | Print the effective host configuration. |

The typical loop after registering a source is `cubby gitignore sync`, `cubby link`, then `cubby status` or `cubby doctor`. Selecting other profiles for one run looks like `cubby link --profiles work,personal` or `CUBBY_PROFILES=personal cubby link`.

## Required external programs

Only `cubby lazygit` runs an external program: [lazygit](https://github.com/jesseduffield/lazygit) must be on `PATH`. Every other command runs on its own.

## Configuration

cubby reads `.cubby.toml` from the current directory only, so run it from the host repository root. Each registered source directory must contain a `cubby.toml` that declares at least one profile. Profiles can also be chosen per invocation with `CUBBY_PROFILES`, `--profiles`, or `--profile`; `cubby config --help` documents the full precedence and every field, and `cubby config` prints the values in effect.

```toml
# host .cubby.toml
profiles = ["work"]

[[source]]
name = "dotfiles"
path = "../dotfiles"
```

```toml
# source cubby.toml
profiles = ["work", "personal"]
ignore = ["**/*.draft.*"]
```

## Non-goals

- No `source add`, `source remove`, or `init` commands; edit the TOML files by hand.
- No generic `git` or `exec` command.
- No tagged release binaries and no documented `go install` path.
