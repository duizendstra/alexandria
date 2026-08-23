#!/usr/bin/env python3
"""OKF vault integrity lint — the validator promised by ADR-0002.

docs/ is an OKF v0.2 bundle, so the vault has two kinds of file and the
lint checks each against its own rules.

Concept documents — every .md that is not a reserved filename — carry the
canonical frontmatter schema in docs/08-reference/okf-profile.md:

  * frontmatter present and well-formed
  * all required fields declared
  * uuid is RFC 4122 and unique across the vault
  * status / maturity / diataxis_quadrant / audience / type enums
  * domain matches the folder the document lives in
  * created_at / updated_at are ISO 8601 timestamps
  * relations are {target_uuid, rel_type} maps whose targets resolve

Reserved filenames — index.md and log.md, at any level (OKF §3.1) — are
not concept documents and carry none of that schema (ADR-0003):

  * index.md holds no frontmatter, the one exception being the bundle-root
    docs/index.md, which declares okf_version and nothing else (§8, §12)
  * an index body is sections of listing entries under headings (§8)
  * log.md groups entries under ISO 8601 YYYY-MM-DD headings (§9)

Together those are the three conditions OKF v0.2 §11 defines as bundle
conformance, plus the Alexandria profile on top of them.

Self-contained: parses the vault's YAML subset directly so it runs on any
python3 without third-party packages. Exit code 0 = clean, 1 = findings.
"""

from __future__ import annotations

import datetime
import re
import sys
from pathlib import Path

REQUIRED_FIELDS = [
    "uuid", "title", "domain", "type", "diataxis_quadrant", "status",
    "maturity", "owner", "summary", "created_at", "updated_at",
    "audience", "tags", "relations",
]

ENUMS = {
    "status": {"active", "draft", "proposed", "accepted", "superseded", "deprecated"},
    "maturity": {"seed", "draft", "standard"},
    "diataxis_quadrant": {"tutorial", "how-to", "reference", "explanation"},
    "type": {"guide", "architecture_decision_record"},
}

AUDIENCES = {"public", "internal"}

DOMAIN_FOLDERS = {
    "01-governance": "governance",
    "02-strategy": "strategy",
    "03-architecture": "architecture",
    "04-decisions": "decisions",
    "05-security": "security",
    "06-operations": "operations",
    "07-playbooks": "playbooks",
    "08-reference": "reference",
}

UUID_RE = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

# OKF §3.1: these names mean something at every level of the hierarchy and
# are never concept documents.
RESERVED = {"index.md", "log.md"}

OKF_VERSION_RE = re.compile(r"^\d+\.\d+$")
ISO_DATE_RE = re.compile(r"^\d{4}-\d{2}-\d{2}$")
HEADING_RE = re.compile(r"^#{1,6}\s+\S")
LIST_LINK_RE = re.compile(r"^\s*[*+-]\s+\[[^]]+\]\([^)]+\)")
LOG_SECTION_RE = re.compile(r"^##\s+(.+?)\s*$")


def strip_scalar(raw: str) -> str:
    """Strip an inline comment and surrounding quotes from a scalar."""
    raw = raw.strip()
    # Inline comments: none of the vault's values contain a bare " #".
    if " #" in raw:
        raw = raw.split(" #", 1)[0].rstrip()
    if len(raw) >= 2 and raw[0] == raw[-1] and raw[0] in "\"'":
        raw = raw[1:-1]
    return raw


def parse_inline_list(raw: str) -> list[str]:
    inner = raw.strip()[1:-1].strip()
    if not inner:
        return []
    return [strip_scalar(part) for part in inner.split(",")]


def parse_frontmatter(lines: list[str], path: str, errors: list[str]) -> dict:
    """Parse the vault's YAML subset: scalars, folded scalars, inline lists,
    and block lists of scalars or flat maps."""
    fields: dict = {}
    i = 0
    while i < len(lines):
        line = lines[i]
        if not line.strip() or line.lstrip().startswith("#"):
            i += 1
            continue
        m = re.match(r"^([A-Za-z_][A-Za-z0-9_]*):(.*)$", line)
        if not m:
            errors.append(f"{path}: unparseable frontmatter line: {line!r}")
            i += 1
            continue
        key, rest = m.group(1), m.group(2).strip()
        if rest == ">" or rest == "|":
            block = []
            i += 1
            while i < len(lines) and (not lines[i].strip() or lines[i].startswith("  ")):
                block.append(lines[i].strip())
                i += 1
            fields[key] = " ".join(b for b in block if b)
            continue
        if rest.startswith("["):
            fields[key] = parse_inline_list(rest)
            i += 1
            continue
        if rest == "":
            # Block list: scalars ("- x") or flat maps ("- k: v" + continuations).
            items: list = []
            i += 1
            while i < len(lines):
                nxt = lines[i]
                if re.match(r"^\s+-\s", nxt):
                    item_line = nxt.strip()[2:]
                    km = re.match(r"^([A-Za-z_][A-Za-z0-9_]*):\s*(.*)$", item_line)
                    if km:
                        entry = {km.group(1): strip_scalar(km.group(2))}
                        i += 1
                        while i < len(lines) and re.match(r"^\s+[A-Za-z_]", lines[i]) \
                                and not re.match(r"^\s+-\s", lines[i]):
                            cm = re.match(r"^\s+([A-Za-z_][A-Za-z0-9_]*):\s*(.*)$", lines[i])
                            if not cm:
                                break
                            entry[cm.group(1)] = strip_scalar(cm.group(2))
                            i += 1
                        items.append(entry)
                    else:
                        items.append(strip_scalar(item_line))
                        i += 1
                else:
                    break
            fields[key] = items
            continue
        fields[key] = strip_scalar(rest)
        i += 1
    return fields


def check_timestamp(value, field: str, path: str, errors: list[str]) -> None:
    if not isinstance(value, str):
        errors.append(f"{path}: {field} must be a string timestamp")
        return
    try:
        datetime.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        errors.append(f"{path}: {field} {value!r} is not ISO 8601")


def split_frontmatter(text: str) -> tuple[bool, str | None, str]:
    """Split a leading `---` block off the body.

    Returns (terminated, block, body). block is None when the file carries
    no frontmatter at all; terminated is False when an opening delimiter is
    never closed. The search starts at the newline that ends the opening
    delimiter, so an empty block (`---` immediately followed by `---`) reads
    as a block with no keys rather than as an unterminated one.
    """
    if not text.startswith("---\n"):
        return True, None, text
    try:
        end = text.index("\n---", 3)
    except ValueError:
        return False, None, text
    rest = text[end + 1:]
    newline = rest.find("\n")
    return True, text[4:end], rest[newline + 1:] if newline != -1 else ""


def check_index(rel: str, text: str, is_root: bool, errors: list[str]) -> None:
    """OKF §8/§12: an index carries no frontmatter, except the bundle root's,
    which declares okf_version and nothing else."""
    terminated, block, body = split_frontmatter(text)
    if not terminated:
        errors.append(f"{rel}: unterminated frontmatter")
        return
    if block is None:
        if is_root:
            errors.append(
                f"{rel}: the bundle-root index must declare okf_version "
                "(OKF v0.2 §12)")
    elif not is_root:
        errors.append(
            f"{rel}: index files carry no frontmatter (OKF v0.2 §8) — only "
            "the bundle-root docs/index.md may declare okf_version")
    else:
        fields = parse_frontmatter(block.splitlines(), rel, errors)
        extra = sorted(set(fields) - {"okf_version"})
        if extra:
            errors.append(
                f"{rel}: bundle-root index frontmatter holds okf_version and "
                f"nothing else (OKF v0.2 §12); also found {extra}")
        version = fields.get("okf_version")
        if not isinstance(version, str) or not OKF_VERSION_RE.match(version):
            errors.append(
                f"{rel}: okf_version {version!r} is not a <major>.<minor> "
                "string (OKF v0.2 §12)")

    lines = body.splitlines()
    if not any(HEADING_RE.match(line) for line in lines):
        errors.append(
            f"{rel}: index body groups its entries under headings "
            "(OKF v0.2 §8); found none")
    if not any(LIST_LINK_RE.match(line) for line in lines):
        errors.append(
            f"{rel}: index body lists no linked entries (OKF v0.2 §8); an "
            "index that would list nothing should not exist")


def check_log(rel: str, text: str, errors: list[str]) -> None:
    """OKF §9: log entries are grouped under ISO 8601 date headings."""
    terminated, _, body = split_frontmatter(text)
    if not terminated:
        errors.append(f"{rel}: unterminated frontmatter")
        return
    dated = 0
    for line in body.splitlines():
        section = LOG_SECTION_RE.match(line)
        if not section:
            continue
        if ISO_DATE_RE.match(section.group(1)):
            dated += 1
        else:
            errors.append(
                f"{rel}: log section heading {section.group(1)!r} is not an "
                "ISO 8601 YYYY-MM-DD date (OKF v0.2 §9)")
    if not dated:
        errors.append(
            f"{rel}: log body has no YYYY-MM-DD section headings "
            "(OKF v0.2 §9)")


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    docs = root / "docs"
    errors: list[str] = []
    uuids: dict[str, str] = {}
    documents: list[tuple[str, dict]] = []
    reserved = 0

    if not (docs / "index.md").is_file():
        errors.append(
            "docs/index.md: the bundle root must carry an index declaring "
            "okf_version (OKF v0.2 §12)")

    for md in sorted(docs.rglob("*.md")):
        rel = md.relative_to(root).as_posix()
        text = md.read_text(encoding="utf-8")

        if md.name in RESERVED:
            reserved += 1
            if md.name == "index.md":
                check_index(rel, text, md.parent == docs, errors)
            else:
                check_log(rel, text, errors)
            continue

        terminated, block, _ = split_frontmatter(text)
        if not terminated:
            errors.append(f"{rel}: unterminated frontmatter")
            continue
        if block is None:
            errors.append(f"{rel}: missing frontmatter (must start with ---)")
            continue
        fields = parse_frontmatter(block.splitlines(), rel, errors)
        documents.append((rel, fields))

        for field in REQUIRED_FIELDS:
            if field not in fields:
                errors.append(f"{rel}: missing required field {field!r}")

        uid = fields.get("uuid")
        if isinstance(uid, str):
            if not UUID_RE.match(uid.lower()):
                errors.append(f"{rel}: uuid {uid!r} is not RFC 4122")
            elif uid in uuids:
                errors.append(f"{rel}: duplicate uuid {uid} (also in {uuids[uid]})")
            else:
                uuids[uid] = rel

        for field, allowed in ENUMS.items():
            val = fields.get(field)
            if isinstance(val, str) and val not in allowed:
                errors.append(
                    f"{rel}: {field} {val!r} not in {sorted(allowed)}")

        aud = fields.get("audience")
        if isinstance(aud, list):
            for a in aud:
                if a not in AUDIENCES:
                    errors.append(f"{rel}: audience {a!r} not in {sorted(AUDIENCES)}")

        folder = md.relative_to(docs).parts[0] if len(md.relative_to(docs).parts) > 1 else None
        if folder in DOMAIN_FOLDERS:
            want = DOMAIN_FOLDERS[folder]
            got = fields.get("domain")
            if isinstance(got, str) and got != want:
                errors.append(f"{rel}: domain {got!r} does not match folder ({want!r})")

        for field in ("created_at", "updated_at"):
            if field in fields:
                check_timestamp(fields[field], field, rel, errors)

        tags = fields.get("tags")
        if "tags" in fields and not isinstance(tags, list):
            errors.append(f"{rel}: tags must be a list")

    # Relations resolve only after every uuid is collected.
    for rel, fields in documents:
        relations = fields.get("relations")
        if relations is None:
            continue
        if not isinstance(relations, list):
            errors.append(f"{rel}: relations must be a list")
            continue
        for item in relations:
            if not isinstance(item, dict) or set(item) != {"target_uuid", "rel_type"}:
                errors.append(
                    f"{rel}: relation {item!r} must be a "
                    "{{target_uuid, rel_type}} map")
                continue
            if item["target_uuid"] not in uuids:
                errors.append(
                    f"{rel}: relation target_uuid {item['target_uuid']} "
                    "does not resolve to any vault document")

    if errors:
        for e in errors:
            print(f"::error::{e}" if "--github" in sys.argv else e)
        print(f"\nokf-lint: {len(errors)} finding(s) across "
              f"{len(documents)} concept documents and {reserved} reserved files")
        return 1

    print(f"okf-lint: {len(documents)} concept documents and "
          f"{reserved} reserved files clean")
    return 0


if __name__ == "__main__":
    sys.exit(main())
