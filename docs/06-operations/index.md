# 06 — Operations

This folder documents the automated pipelines, release tooling, and operational procedures that keep Alexandria healthy.

## What Belongs Here

- **CI/CD Pipeline Design** — GitHub Actions workflows for linting, testing, and publishing.
- **Release-Please Automation** — Configuration and conventions for automated changelog and version bumps.
- **Dependabot Config** — Dependency update schedules and grouping rules.
- **Tagging Conventions** — Module-scoped tag format (e.g., `go/slog-gcp/v0.1.0`).

## Contents

* [Declarative CI/CD Pipelines & Release Automation](declarative-ci-cd-pipelines.md) - Defines the continuous integration rules, release workflows, and semantic tagging standards for multi-module monorepos, distinguishing the pipeline that runs today from planned automation.
* [Disaster Recovery & Release Rollback Playbook](disaster-recovery-and-rollback.md) - Step-by-step operational instructions for hot-fixing, rolling back faulty releases, retracting Go modules, and recovering from pipeline incidents.
