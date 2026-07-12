---
name: ko-build
description: >
  Sets up ko container builds for Go services targeting GCP Cloud Run.
  Uses the Alexandria golden .ko.yaml template with pinned Chainguard base
  image and reproducible build flags. Trigger: "set up ko",
  "add ko build", "container build", "ko configuration", "deploy to Cloud Run".
---

# ko Build for Go Cloud Run Services

Use ko as the container build tool for all pure-Go services deploying to
GCP Cloud Run. ko builds Go source directly into minimal container images
without a Dockerfile or Docker daemon.

## Golden Template

Copy the template from `blueprints/service/.ko.yaml` into the service root.
This path is relative to the Alexandria monorepo root:

```bash
cp blueprints/service/.ko.yaml ./
```

## Template Anatomy

```yaml
# Base image — Chainguard static for minimal attack surface.
# This is ko's own default; being explicit for documentation.
# Pinned to digest for reproducible builds. Update with:
#   crane digest cgr.dev/chainguard/static:latest
defaultBaseImage: cgr.dev/chainguard/static:latest@sha256:<digest>

defaultPlatforms:
  - linux/amd64

# Explicit SBOM generation (ko default, declared for visibility).
sbom: spdx

builds:
  - dir: .
    main: .
    flags:
      - -trimpath        # reproducible builds (strips local paths)
    ldflags:
      - -w               # strip DWARF (saves ~10-15% binary size)
                          # -s intentionally omitted to keep symbol table
                          # for pprof profiling and readable panic traces
    env:
      - CGO_ENABLED=0    # explicit (ko default) — pure-Go static binary
```

OCI labels are applied via CLI flags in CI (not in `.ko.yaml`):
```bash
ko build --image-label org.opencontainers.image.source=https://github.com/OWNER/REPO .
```

## Design Decisions

| Decision | Rationale |
|---|---|
| Chainguard over distroless | ko's own default base; more actively maintained, faster CVE response than distroless |
| No `-buildvcs=false` | Preserves git commit provenance in binary (~200 bytes) |
| No `-s` ldflag | Keeps symbol table for pprof and readable panic stack traces |
| Keep `-w` ldflag | Strips DWARF debug info (rarely needed in prod, saves ~10-15%) |
| Explicit `sbom: spdx` | ko generates SBOMs by default; explicit declaration documents intent |
| Pinned digest | Floating tags break reproducibility silently |
| `linux/amd64` only | Cloud Run runs amd64; add `linux/arm64` when needed |
| No build `id` | Not needed for single-service repos; add for multi-binary repos |
| Labels via CLI | ko `.ko.yaml` does not support `labels` in `builds`; use `--image-label` |

## Required: Timezone Data

The static base image has no `/usr/share/zoneinfo`. Every service
must import the embedded timezone database:

```go
import _ "time/tzdata"
```

## Local Development

```bash
# Build and push to registry (no Docker daemon needed)
export KO_DOCKER_REPO=europe-west1-docker.pkg.dev/PROJECT/REPO
ko build .

# Build to local Docker daemon (requires Colima/Docker)
ko build . --local

# Build to OCI tarball
ko build . --tarball=my-service.tar
```

## CI/CD (GitHub Actions)

```yaml
- uses: ko-build/setup-ko@v0.8

- uses: google-github-actions/auth@v2
  with:
    workload_identity_provider: 'projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL/providers/PROVIDER'
    service_account: 'deploy@PROJECT.iam.gserviceaccount.com'

- name: Build and push
  run: |
    export KO_DOCKER_REPO=europe-west1-docker.pkg.dev/$PROJECT/$REPO
    IMAGE=$(ko build --platform=linux/amd64 \
      --bare --tags=$(git rev-parse --short HEAD) \
      --image-label org.opencontainers.image.source=https://github.com/OWNER/$REPO .)
    echo "image=$IMAGE" >> "$GITHUB_OUTPUT"
```

Key flags:
- `--bare`: Use repo path as image name (no hash suffix)
- `--tags=<git-sha>`: Reproducible tags instead of `:latest`
- `--image-label`: OCI metadata (applied via CLI, not `.ko.yaml`)

No Docker daemon needed on the runner. No service account JSON keys.

## Updating the Base Image Digest

```bash
# Get latest digest
crane digest cgr.dev/chainguard/static:latest

# Update .ko.yaml (portable across macOS and Linux)
NEW_DIGEST=$(crane digest cgr.dev/chainguard/static:latest)
sed -i.bak "s/@sha256:.*/@sha256:$NEW_DIGEST/" .ko.yaml && rm -f .ko.yaml.bak
```

Use Renovate or Dependabot to automate this.

## When NOT to Use ko

Fall back to a Dockerfile when:

- The service requires CGO (SQLite, libvips, etc.)
- The image needs OS-level packages (fonts, native libs)
- The service is not written in Go
- Custom entrypoint scripts or init processes are needed

## Reference

- [ko.build documentation](https://ko.build)
- [ko GitHub](https://github.com/ko-build/ko)
- [setup-ko action](https://github.com/ko-build/setup-ko)
- [Chainguard static image](https://images.chainguard.dev/directory/image/static/overview)
