#!/usr/bin/env python3
"""Sync the README module-index version column from .release-please-manifest.json.

The root README's module index carries a version cell per module. Releases bump
.release-please-manifest.json, and CI's "Module hygiene" job fails until the
README row matches. Running this script rewrites every version cell to the
manifest value; the release workflow runs it against the release-please PR
branch so the bump lands in the release PR itself.
"""

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
IMPORT_RE = re.compile(r"`github\.com/duizendstra/alexandria/([^`]+)`")


def main() -> int:
    manifest = json.loads((ROOT / ".release-please-manifest.json").read_text())
    readme = ROOT / "README.md"
    text = readme.read_text()

    out = []
    changed = []
    for line in text.split("\n"):
        cells = line.split("|")
        if len(cells) >= 5:
            match = IMPORT_RE.search(cells[2])
            if match and match.group(1) in manifest:
                want = f" v{manifest[match.group(1)]} "
                if cells[3] != want:
                    changed.append(f"{match.group(1)} ->{want.rstrip()}")
                    cells[3] = want
        out.append("|".join(cells))

    new = "\n".join(out)
    if new != text:
        readme.write_text(new)

    if changed:
        print("synced: " + ", ".join(changed))
    else:
        print("README already in sync")

    return 0


if __name__ == "__main__":
    sys.exit(main())
