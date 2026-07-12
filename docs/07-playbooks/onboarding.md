---
uuid: fa14b87a-a4b4-454c-ad8f-a75a201d19b9
title: "Developer Onboarding Playbook"
domain: "playbooks"
type: "guide"
diataxis_quadrant: "tutorial"
status: "active"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-03-04T09:00:00Z"
updated_at: "2026-07-12T14:30:00Z"
summary: >
  Step-by-step developer learning playbook for Alexandria, achieving a 60-second
  local development setup using Nix, nix-direnv, and Colima.
audience: [public]
tags: [ "onboarding", "playbook", "nix" ]
relations:
  - target_uuid: "91bc6c63-db3d-4c31-90be-e0dfc3df2220"
    rel_type: "depends_on"
---
# Developer Onboarding Playbook

Welcome to **Alexandria**, the authoritative shared software repository of our platform ecosystem. 

This playbook guides you through our **60-second declarative onboarding protocol**, taking you from a fresh repository clone to running green integration tests on your local machine.

---

## 🛠️ Workstation Prerequisites

We treat our local engineering workstations with identical rigor to our production systems. Our development toolchains are entirely declarative and isolated. **Do not install Go, Protobuf, Buf, GCloud, or Taskfiles manually.**

1.  **Identity Access** — Confirm you have been added to the appropriate identity access groups and hold active GitHub read permissions.
2.  **Nix Package Manager** — We use Nix to manage identical hermetic environments across macOS and Linux:
    ```bash
    # Install the multi-user Nix package manager (Recommended)
    curl -L https://nixos.org/nix/install | sh
    ```
3.  **Direnv Shell Loader** — Used to automatically load the Nix environment upon entering project directories:
    ```bash
    # Install Direnv
    brew install direnv # on macOS
    # Or your standard package manager on Linux
    ```

---

## 🚀 The 60-Second Boot Protocol

Open your terminal and execute these three commands:

### Step 1: Clone the Ecosystem Repository
```bash
git clone git@github.com:OWNER/alexandria.git
cd alexandria
```

### Step 2: Establish the Nix Hermetic Shell
The repository contains a `flake.nix` file mapping all dependencies (pinned Go versions, linters, Buf CLI, and test runners).
```bash
# Allow direnv to load the Nix flake automatically
direnv allow
```
*   *Note: On your first run, Nix downloads and caches all required dependencies. This occurs once. Subsequent directory entries are instant (under 1 second).*

### Step 3: Run the Local Integration Tests
Verify that you can run automated checks locally in a module directory (e.g., `go/google/`):
```bash
cd go/google
GOWORK=off go test -v ./...
```

---

## 🛠️ Typical Development Lifecycle

When working on Alexandria modules:

1.  **Synchronize Your Work** — Always branch off a clean, up-to-date `main` branch.
2.  **Define Your Contract First** — If changing service payloads, edit or write the Protobuf contracts in `contracts/proto/` first, and run `buf generate`.
3.  **Harden Your Code** — Ensure your Go changes pass our SRE performance metrics (minimal allocations on hot paths, proper Keep-Alive socket connection recycling).
4.  **Validate Metadata** — Ensure all links and OKF metadata files are valid before submitting.
5.  **Submit a Pull Request** — Code is merged to `main` using squash merges to preserve history purity. Direct commits to `main` are blocked.
