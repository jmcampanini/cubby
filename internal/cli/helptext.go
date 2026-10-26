package cli

// Shared help fragments compose command long descriptions so repeated
// contract text cannot drift between commands. Fragments carry no leading or
// trailing newline; compose them with explicit separators.

const hostRootHelp = `cubby reads .cubby.toml from the current directory and treats that
directory as the host repository root. It does not search parent
directories or any user-level config location, so run it from the host
root.`

const profileSelectionHelp = `Profiles come from the highest layer that sets them: --profiles LIST
(comma-separated) and --profile NAME, which may repeat and combine into
one list, override CUBBY_PROFILES, which overrides profiles in
.cubby.toml. Entries are trimmed and blanks dropped. CUBBY_PROFILES and
flags keep the last occurrence of each duplicate in its position. Flag
values combine all --profiles entries before all --profile entries,
regardless of their order on the command line, before removing duplicates.
TOML profiles keep the first occurrence. When env_profiles names an
environment variable that is set, its comma-separated value is appended
last, keeping the first occurrence across the combined list. cubby keeps
no active-profile state; 'cubby profile effective' prints the list any
command would use.`

const profileFileGrammarHelp = `A source file is profile-scoped when its basename contains .<profile> as a
whole dot-separated segment and is not exactly .<profile>: zshrc.work,
nvim/init.work.lua, git/config.personal.toml. cubby walks every
registered source directory for such files. Only regular files count;
.git directories, cubby.toml, and paths matched by the source's ignore
patterns are skipped. The host path keeps the same relative path and
filename, suffix included.`

const undeclaredProfileNoticeHelp = `For each selected profile that a source does not declare, a notice
'source "<name>" does not declare selected profile "<profile>"; skipping'
goes to stderr and that source contributes no files for it. A selected
profile that no registered source declares is an error.`

const managedLinkHelp = `cubby walks the host root, skipping .git and source directories nested in
the host, for managed links: symlinks whose target resolves inside a
registered source directory. Symlinks pointing anywhere else are ignored.`

const driftReasonsHelp = `Drift reasons: dangling (target missing), unresolved target (target
cannot be resolved), non-regular target (target is not a regular file),
path mismatch (host and source relative paths differ), unknown profile
(basename matches no profile the source declares), and ignored (the
source's ignore patterns match it).`

const gitignorePatternsHelp = `The required patterns are /.cubby.toml plus '*.<profile>.*' and
'*.<profile>' for every profile in the union of profiles declared by
registered sources. cubby reads .gitignore at the host root; a pattern
counts as present only when it appears as an exact trimmed line that is
not a comment, and a missing .gitignore counts as empty. All registered
sources must load.`

const noPromptHelp = `This command never prompts and works without a terminal.`

const jsonContractHelp = `--json writes one JSON object on a single line to stdout and nothing else
to stdout; diagnostics still go to stderr, the output contains no
terminal escapes, and the exit status is the same as without --json.`

const colorModeHelp = `--color decides whether stdout is styled: auto (the default) styles it
only when stdout is a terminal, TERM is not dumb, and NO_COLOR is unset
or empty; always and never override that detection; --json output never
carries escapes.`

const commandGroupHelp = `Run without a subcommand this prints help on stdout and reads nothing;
each subcommand's --help states its own contract.`
