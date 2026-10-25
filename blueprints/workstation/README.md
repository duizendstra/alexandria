# Workstation

Developer workstation bootstrap scripts for the `pass` + GPG secrets workflow.

## Scripts

| Script | Description |
|---|---|
| [bootstrap-gpg.sh](bootstrap-gpg.sh) | Configures gpg-agent for non-interactive use: enables `allow-preset-passphrase`, restarts the agent, and presets the passphrase for every secret key so `pass`, git signing, and encrypted-state tooling stop prompting |
| [load-secrets.sh](load-secrets.sh) | Reads a `.secrets.yaml` manifest mapping env var names to `pass` entries and prints `export` statements with the decrypted values — `eval "$(./load-secrets.sh)"` |

## Workflow

1. Store credentials in [pass](https://www.passwordstore.org/).
2. Declare the ones a workspace needs in a gitignored `.secrets.yaml`:

   ```yaml
   secrets:
     GITHUB_TOKEN: "github/mytoken"
     API_KEY: "vendor/api-key"
   ```

3. Unlock the agent once per session: `./bootstrap-gpg.sh` prompts for the
   passphrase, or reads it from a pipe. A passphrase given as an argument still
   works, but every user on the machine can read it in the process list while
   the script runs, and it stays in shell history, so the script warns.
4. Load the secrets into the shell: `eval "$(./load-secrets.sh)"`. Keep the
   quotes. The output is built to survive the unquoted form too (below), but
   the quoted form is the one to rely on.

Decryption runs with `--pinentry-mode error`, so a locked agent produces a
warning instead of a hidden pinentry prompt — failures are visible and scriptable.

Every value, path and message is printed base64-encoded and decoded by the
shell that runs `eval`, so `eval` assigns a value and never runs it, with or
without the quotes. A secret containing `$`, a backtick, a quote or a `*`
arrives byte for byte. Decoding needs `base64 -d` (coreutils, or macOS 13 and
later). Indent the manifest's entries by any amount; the block ends at the next
line that starts in column 0.
