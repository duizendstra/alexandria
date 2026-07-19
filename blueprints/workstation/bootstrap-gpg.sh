#!/usr/bin/env bash
# bootstrap-gpg.sh — Automates gpg-agent configuration and non-interactive passphrase caching.
#
# Configures allow-preset-passphrase, restarts the agent, and presets the
# passphrase for every secret key in the keyring so that pass, git signing,
# and encrypted-state tooling work without pinentry prompts.
#
# Usage:
#   ./bootstrap-gpg.sh [passphrase]

set -euo pipefail

PASSPHRASE="${1:-}"

# 1. Ensure allow-preset-passphrase is in agent config
mkdir -p "$HOME/.gnupg"
chmod 700 "$HOME/.gnupg"
AGENT_CONF="$HOME/.gnupg/gpg-agent.conf"

if [ ! -f "$AGENT_CONF" ] || ! grep -q "allow-preset-passphrase" "$AGENT_CONF"; then
  echo "allow-preset-passphrase" >> "$AGENT_CONF"
  chmod 600 "$AGENT_CONF"
  echo "✔ Added allow-preset-passphrase to $AGENT_CONF"
fi

# 2. Restart/reload gpg-agent to apply changes
gpgconf --kill gpg-agent 2>/dev/null || true
gpg-agent --daemon --use-standard-socket >/dev/null 2>&1
echo "✔ Started/Reloaded gpg-agent daemon"

# 3. Resolve passphrase
if [ -z "$PASSPHRASE" ]; then
  # If the agent can already sign non-interactively, there is nothing to do
  if echo test | gpg --batch --pinentry-mode error --sign >/dev/null 2>&1; then
    echo "✔ Passphrase is already cached and the agent is operational. No action needed."
    exit 0
  fi
  # Otherwise, prompt the user
  read -rs -p "Enter PGP private key passphrase: " PASSPHRASE
  echo
fi

# 4. Find all secret key keygrips
KEYGRIPS=()
while read -r line; do
  if [ -n "$line" ]; then
    KEYGRIPS+=("$line")
  fi
done < <(gpg --list-secret-keys --with-keygrip 2>/dev/null | awk '/Keygrip =/ {print $3}' || true)

if [ ${#KEYGRIPS[@]} -eq 0 ]; then
  echo "⚠ No secret keys found in the GPG keyring."
  exit 0
fi

# 5. Resolve gpg-preset-passphrase binary — gpgconf knows, fall back to common installs
PRESET_BIN=""
LIBEXEC_DIR="$(gpgconf --list-dirs libexecdir 2>/dev/null || true)"
SEARCH_PATHS=(
  "${LIBEXEC_DIR:+$LIBEXEC_DIR/gpg-preset-passphrase}"
  "/opt/homebrew/opt/gnupg/libexec/gpg-preset-passphrase"
  "$HOME/homebrew/opt/gnupg/libexec/gpg-preset-passphrase"
  "/usr/lib/gnupg/gpg-preset-passphrase"
  "/usr/libexec/gpg-preset-passphrase"
)
for path in "${SEARCH_PATHS[@]}"; do
  if [ -n "$path" ] && [ -x "$path" ]; then
    PRESET_BIN="$path"
    break
  fi
done

if [ -z "$PRESET_BIN" ]; then
  echo "ERROR: gpg-preset-passphrase binary not found. Please ensure gnupg is fully installed." >&2
  exit 1
fi

# 6. Preset all keygrips
echo "Caching passphrase for ${#KEYGRIPS[@]} keygrips..."
for grip in "${KEYGRIPS[@]}"; do
  echo "$PASSPHRASE" | "$PRESET_BIN" --preset "$grip"
done

echo "✔ GPG Agent cached all keys successfully!"
