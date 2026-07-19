# Workstation

Developer workstation bootstrap scripts for the `pass` + GPG secrets workflow.

## Scripts

| Script | Description |
|---|---|
| [bootstrap-gpg.sh](bootstrap-gpg.sh) | Configures gpg-agent for non-interactive use: enables `allow-preset-passphrase`, restarts the agent, and presets the passphrase for every secret key so `pass`, git signing, and encrypted-state tooling stop prompting |
| [load-secrets.sh](load-secrets.sh) | Reads a `.secrets.yaml` manifest mapping env var names to `pass` entries and prints `export` statements with the decrypted values — `eval $(./load-secrets.sh)` |

## Workflow

1. Store credentials in [pass](https://www.passwordstore.org/).
2. Declare the ones a workspace needs in a gitignored `.secrets.yaml`:

   ```yaml
   secrets:
     GITHUB_TOKEN: "github/mytoken"
     API_KEY: "vendor/api-key"
   ```

3. Unlock the agent once per session: `./bootstrap-gpg.sh`
4. Load the secrets into the shell: `eval $(./load-secrets.sh)`

Decryption runs with `--pinentry-mode error`, so a locked agent produces a
warning instead of a hidden pinentry prompt — failures are visible and scriptable.
