#!/usr/bin/env bash
# load-secrets.sh — Parses a .secrets.yaml manifest and prints export statements
# with credentials decrypted from pass.
#
# The manifest maps environment variable names to pass entries:
#
#   secrets:
#     GITHUB_TOKEN: "github/mytoken"
#     API_KEY: "vendor/api-key"
#
# Usage:
#   eval $(./load-secrets.sh [path/to/.secrets.yaml])
#
# Defaults to ./.secrets.yaml. Decryption is non-interactive: if the GPG agent
# is locked, a warning is printed instead of a pinentry prompt (see bootstrap-gpg.sh).

SECRETS_FILE="${1:-.secrets.yaml}"

if [ ! -f "$SECRETS_FILE" ]; then
  echo "echo \"[ERROR] Secrets manifest not found: $SECRETS_FILE\" >&2"
  exit 1
fi

SECRETS_FILE="$SECRETS_FILE" python3 << 'EOF'
import os, re, subprocess

with open(os.environ["SECRETS_FILE"]) as f:
    in_secrets = False
    for line in f:
        if "secrets:" in line:
            in_secrets = True
            continue
        if in_secrets:
            # If line is less indented than the secrets entries, we are done
            if line.strip() and not line.startswith("    "):
                break
            match = re.match(r"^\s+([A-Z0-9_]+):\s*\"?(.*?)\"?\s*$", line)
            if match:
                key, pass_path = match.groups()
                try:
                    env = os.environ.copy()
                    env["PASSWORD_STORE_GPG_OPTS"] = "--batch --pinentry-mode error"
                    val = subprocess.check_output(
                        ["pass", "show", pass_path],
                        stderr=subprocess.PIPE, env=env,
                    ).decode("utf-8").strip()
                    print(f"export {key}=\"{val}\"")
                except subprocess.CalledProcessError:
                    print(f"echo \"[WARN] Could not decrypt {pass_path}. Is GPG locked? Run: echo 'test' | gpg --sign\" >&2")
                except Exception as e:
                    print(f"echo \"[WARN] Error accessing pass for {pass_path}: {e}\" >&2")
EOF
