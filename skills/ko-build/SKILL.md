---
name: ko-build
description: >
  Sets up ko container builds for Go services targeting GCP Cloud Run.
  Uses the Alexandria golden .ko.yaml template with pinned distroless base
  image, reproducible build flags, and OCI labels. Trigger: "set up ko",
  "add ko build", "container build", "ko configuration", "deploy to Cloud Run".
---

# Ko Build for Go Cloud Run Services

Use ko as the container build tool for all pure-Go services deploying to
GCP Cloud Run. Ko builds Go source directly into minimal distroless images
without a Dockerfile or Docker daemon.

## Golden Template

Copy the template from `blueprints/service/.ko.yaml` into the service root:

```bash
cp blueprints/service/.ko.yaml ./
```

Then update the OCI source label to match the repo.

## Template Anatomy

```yaml
# Base image — pinned to digest for reproducible builds
# Update digest: crane digest gcr.io/distroless/static-debian13:nonroot
defaultBaseImage: gcr.io/distroless/static-debian13:nonroot@sha256:<digest>

defaultPlatforms:
  - linux/amd64

builds:
  - dir: .
    main: .
    flags:
      - -trimpath        # reproducible builds (strips local paths)
    ldflags:
      - -w               # strip DWARF (saves ~15-20% binary size)
                          # -s intentionally omitted to keep symbol table
                          # for pprof profiling and readable panic traces
    env:
      - CGO_ENABLED=0    # pure-Go static binary
    labels:
      org.opencontainers.image.source: https://github.com/duizendstra-com/REPO
```

## Design Decisions

| Decision | Rationale |
|---|---|
| No `-buildvcs=false` | Preserves git commit provenance in binary (~200 bytes) |
| No `-s` ldflag | Keeps symbol table for pprof and readable panic stack traces |
| Keep `-w` ldflag | Strips DWARF debug info (rarely needed in prod, saves ~15-20%) |
| Pinned digest | Floating tags break reproducibility silently |
| `linux/amd64` only | Cloud Run runs amd64; add `linux/arm64` when needed |
| No build `id` | Not needed for single-service repos; add for multi-binary repos |

## Required: Timezone Data

The static distroless image has no `/usr/share/zoneinfo`. Every service
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
    IMAGE=$(ko build --platform=linux/amd64 .)
    echo "image=$IMAGE" >> "$GITHUB_OUTPUT"
```

No Docker daemon needed on the runner. No service account JSON keys.

## Updating the Base Image Digest

```bash
# Get latest digest
crane digest gcr.io/distroless/static-debian13:nonroot

# Update .ko.yaml
sed -i '' "s/@sha256:.*/@sha256:$(crane digest gcr.io/distroless/static-debian13:nonroot)/" .ko.yaml
```

Use Renovate or Dependabot to automate this.

## When NOT to Use Ko

Fall back to a Dockerfile when:

- The service requires CGO (SQLite, libvips, etc.)
- The image needs OS-level packages (fonts, native libs)
- The service is not written in Go
- Custom entrypoint scripts or init processes are needed

## Reference

- [ko.build documentation](https://ko.build)
- [ko GitHub](https://github.com/ko-build/ko)
- [setup-ko action](https://github.com/ko-build/setup-ko)
- [distroless images](https://github.com/GoogleContainerTools/distroless)
