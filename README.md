# cubby

cubby layers profile-scoped dotfiles from one or more source repositories into a host repository as relative symlinks. A host repository registers its sources in `.cubby.toml`, each source declares its profiles in `cubby.toml`, and a file whose name carries a profile suffix (`zshrc.work`, `nvim/init.work.lua`) is linked into the host at the same relative path whenever that profile is selected. cubby never replaces an existing host file and never accesses the network.


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

With Git and [mise](https://mise.jdx.dev) installed (`brew install mise` on macOS), mise supplies the pinned Go toolchain from `mise.toml`:

```sh
git clone https://github.com/jmcampanini/cubby
cd cubby
mise trust
mise install
mise run build
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
| `cubby profile effective [--json]` | Print the profiles this invocation would use. |
| `cubby source list [--json]` | List registered sources. |
| `cubby lazygit [--source NAME]` | Open lazygit in a source repository. |

Command help is the canonical reference: `cubby --help`, each command's `--help`, and the help topics `cubby help workflow` and `cubby help exit-codes` describe every user-facing contract.

The typical loop after registering a source is `cubby gitignore sync`, `cubby link`, then `cubby status` or `cubby doctor`.


## Required external programs

- [lazygit](https://github.com/jesseduffield/lazygit) must be on `PATH` for `cubby lazygit`, which hands it the terminal.


## Configuration

cubby reads `.cubby.toml` from the current directory only, so run it from the host repository root. Each registered source directory must contain a `cubby.toml`. `cubby config --help` documents the load order, profile selection, and every field in both files.

```toml
# host .cubby.toml
profiles = ["work"]

[[source]]
name = "dotfiles"
path = "../dotfiles"
```
