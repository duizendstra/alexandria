# Alexandria task runner — the local mirror of the CI matrix.
#
# Every *-all recipe iterates `find go -name go.mod`, exactly like the CI
# detect-modules job, so local runs and CI cannot diverge on module coverage.
# GOWORK=off matches CI: modules resolve standalone via published pins.

# List available recipes.
default:
    @just --list

# Run go test -race across every module.
test-all:
    #!/usr/bin/env sh
    set -e
    for modfile in $(find go -name go.mod | sort); do
        dir=$(dirname "$modfile")
        echo "==> test $dir"
        (cd "$dir" && GOWORK=off go test -race -count=1 ./...)
    done

# Run go vet across every module.
vet-all:
    #!/usr/bin/env sh
    set -e
    for modfile in $(find go -name go.mod | sort); do
        dir=$(dirname "$modfile")
        echo "==> vet $dir"
        (cd "$dir" && GOWORK=off go vet ./...)
    done

# Run golangci-lint across every module.
lint-all:
    #!/usr/bin/env sh
    set -e
    for modfile in $(find go -name go.mod | sort); do
        dir=$(dirname "$modfile")
        echo "==> lint $dir"
        (cd "$dir" && GOWORK=off golangci-lint run ./...)
    done

# Print per-module test coverage (excluding generated go/contracts).
cover-all:
    #!/usr/bin/env sh
    set -e
    GOWORK=off
    export GOWORK
    for modfile in $(find go -name go.mod | sort); do
        dir=$(dirname "$modfile")
        case "$dir" in go/contracts) continue ;; esac
        (cd "$dir" && go test -count=1 -coverprofile=/tmp/alx-cover.out ./... >/dev/null 2>&1 \
            && printf '%-45s %s\n' "$dir" "$(go tool cover -func=/tmp/alx-cover.out | tail -1 | awk '{print $NF}')" \
            || printf '%-45s FAIL\n' "$dir")
    done

# vet + lint + test everything — the full pre-push gate.
check: vet-all lint-all test-all
