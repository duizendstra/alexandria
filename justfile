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

# Run native Go fuzz testing (fuzztime=2s) across every package with fuzz targets.
fuzz-all:
    #!/usr/bin/env sh
    set -e
    for file in $(find go -name '*_test.go' | sort); do
        grep -o '^func Fuzz[A-Za-z0-9_]*' "$file" | sed 's/^func //' | while read -r fn; do
            [ -z "$fn" ] && continue
            dir=$(dirname "$file")
            echo "==> fuzz $dir :: $fn"
            (cd "$dir" && GOWORK=off go test -fuzz="^${fn}$" -fuzztime=2s .)
        done
    done

# Modernize code across every module using go fix.
modernize:
    #!/usr/bin/env sh
    set -e
    for modfile in $(find go -name go.mod | sort); do
        dir=$(dirname "$modfile")
        echo "==> modernize $dir"
        (cd "$dir" && GOWORK=off go fix ./...)
    done

# Verify release manifest coverage and README version parity across every module.
check-versions:
    #!/usr/bin/env sh
    set -e
    for modfile in $(find go -name go.mod | sort); do
        dir=$(dirname "$modfile")
        if ! jq -e --arg m "$dir" 'has($m)' .release-please-manifest.json > /dev/null; then
            echo "Missing manifest entry for $dir"
            exit 1
        fi
        if ! jq -e --arg m "$dir" '.packages | has($m)' .release-please-config.json > /dev/null; then
            echo "Missing packages entry for $dir"
            exit 1
        fi
        manifest_ver=$(jq -r --arg m "$dir" '.[$m]' .release-please-manifest.json)
        readme_ver=$(grep "\`github.com/duizendstra/alexandria/$dir\`" README.md | awk -F '|' '{print $4}' | tr -d ' v')
        if [ "$manifest_ver" != "$readme_ver" ]; then
            echo "Module $dir version mismatch: manifest=$manifest_ver, README=$readme_ver"
            exit 1
        fi
    done
    echo "All Go modules in sync with release manifest and README."

# vet + lint + test + fuzz + version check — the full pre-push gate.
check: vet-all lint-all test-all fuzz-all check-versions

