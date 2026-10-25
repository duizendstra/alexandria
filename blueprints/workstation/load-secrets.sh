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
#   eval "$(./load-secrets.sh [path/to/.secrets.yaml])"
#
# Every value, path and message is printed base64-encoded and decoded by the
# shell that evals it, so the output holds no whitespace run, newline, glob
# character or quote taken from a secret. eval assigns a value and never runs
# it, whether or not the substitution is quoted: a secret containing $, a
# backtick, a quote or a * arrives byte for byte. Quote it anyway.
#
# Defaults to ./.secrets.yaml. Decryption is non-interactive: if the GPG agent
# is locked, a warning is printed instead of a pinentry prompt (see bootstrap-gpg.sh).

SECRETS_FILE="${1:-.secrets.yaml}"

if [ ! -f "$SECRETS_FILE" ]; then
  msg="[ERROR] Secrets manifest not found: $SECRETS_FILE"
  printf "printf '%%s\\\\n' \"\$(printf '%%s' '%s' | base64 -d)\" >&2;\n" \
    "$(printf '%s' "$msg" | base64 | tr -d '\n')"
  exit 1
fi

SECRETS_FILE="$SECRETS_FILE" python3 << 'EOF'
import base64, os, re, subprocess

def decoded(text):
    # Base64 output is [A-Za-z0-9+/=] only: nothing in it splits, globs or quotes.
    b64 = base64.b64encode(text.encode("utf-8")).decode("ascii")
    return f"\"$(printf '%s' '{b64}' | base64 -d)\""

def warn(msg):
    print(f"printf '%s\\n' {decoded(msg)} >&2;")

with open(os.environ["SECRETS_FILE"]) as f:
    in_secrets = False
    for line in f:
        if "secrets:" in line:
            in_secrets = True
            continue
        if in_secrets:
            # A line back at column 0 is the next top-level key: we are done.
            if line.strip() and not line[0].isspace():
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
                    print(f"export {key}={decoded(val)};")
                except subprocess.CalledProcessError:
                    warn(f"[WARN] Could not decrypt {pass_path}. Is GPG locked? Run: echo 'test' | gpg --sign")
                except Exception as e:
                    warn(f"[WARN] Error accessing pass for {pass_path}: {e}")
EOF
