---
uuid: cf0867f7-87b0-449c-9c15-54fc13ed895e
title: "macOS Shell Pitfalls for Agent-Driven Automation"
domain: "reference"
type: "guide"
diataxis_quadrant: "reference"
status: "active"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-08-22T09:00:00Z"
updated_at: "2026-08-22T09:00:00Z"
summary: >
  zsh word-splitting and colon modifiers, plus macOS's frozen bash 3.2,
  make ad hoc multi-step shell commands unreliable for automated/agent
  workflows on macOS; write such steps as an explicit bash script instead.
audience: [public]
tags: [ "shell", "macos", "zsh", "bash", "automation", "lessons-learned" ]
relations: []
---

# macOS Shell Pitfalls for Agent-Driven Automation

Automation that generates and runs shell commands (scripted CI steps,
agent-driven tooling) tends to be written and tested against Linux/bash
assumptions. macOS breaks several of those assumptions by default, and the
failures are silent rather than syntax errors.

## Symptom

A multi-step shell operation — copying a computed list of files, building
a path from a variable, chaining a `cd` into a following command — produces
the wrong result with no error: a file list collapses into a single
argument, part of a variable name or path is silently dropped, or a script
using `mapfile`/associative arrays fails or behaves unexpectedly.

## Cause

Three macOS-specific behaviors combine to produce this:

1. **zsh is the default interactive shell**, and unlike bash/sh, zsh does
   **not** word-split an unquoted `$VAR` expansion. A variable holding a
   newline- or space-joined list of paths is passed to the next command as
   one argument, not several.
2. **zsh's `:` modifiers** (`$VAR:s/x/y/`, `$VAR:c`, `$VAR:h`, `$VAR:t`,
   etc.) trigger whenever a colon immediately follows a variable
   expansion — including cases that look like a path or ref separator
   (`$BRANCH:some/path`), silently reinterpreting the text after the colon
   instead of concatenating it.
3. **`/bin/bash` on macOS is frozen at version 3.2** (Apple ships no newer
   bash for licensing reasons), so bash 4+ builtins such as `mapfile` and
   native associative arrays are unavailable even when a script's shebang
   says `#!/bin/bash`.

## Rule

- Always quote variable expansions (`"$VAR"`, `"${VAR}"`), especially
  before array/list use.
- Never place a bare `:` immediately after an unbraced variable expansion;
  write `"${VAR}:path"` (braced) rather than `$VAR:path`.
- For any multi-step operation involving more than one variable or more
  than one command chained with `&&`/`cd`, write it as a standalone
  `#!/usr/bin/env bash` (or explicitly-pinned newer bash) script file and
  invoke that file, rather than composing it inline in the interactive
  shell. This sidesteps both the zsh-vs-bash behavior gap and the
  macOS-bash-3.2 feature gap in one move.
- If a script genuinely needs bash 4+ features, depend on a Homebrew-
  installed bash explicitly (`#!/usr/bin/env bash` plus a version check,
  or an absolute `/opt/homebrew/bin/bash` shebang) rather than assuming
  `/bin/bash` supports them.

## How to Test for It

- Run the same command block with `set -x` in both `zsh` and `bash` and
  diff the traced (expanded) commands — a divergence points at
  word-splitting or a colon-modifier bug.
- Grep scripts and inline command strings for an unbraced variable
  immediately followed by a colon (`\$[A-Za-z_]+:`) as a targeted search
  for the modifier trap.
- Run `bash --version` in the target environment before trusting a script
  that uses `mapfile`, `readarray`, or associative arrays; add an explicit
  version guard at the top of the script if it depends on them.
