#!/usr/bin/env bash
set -euo pipefail

# scripts/release-smoke.sh
# Verifies that a packaged release artifact starts correctly and reports the expected version.

VERSION="${1:-${VERSION:-}}"
DIST_DIR="${2:-dist}"

if [ -z "$VERSION" ]; then
    echo "Error: VERSION must be provided as first argument or environment variable."
    exit 1
fi

# GoReleaser naming convention from .goreleaser.yml:
# {{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}
# Archive format: tar.gz
ARCHIVE="${DIST_DIR}/whooper_${VERSION}_linux_amd64.tar.gz"

echo "Checking for archive: $ARCHIVE"
if [ ! -f "$ARCHIVE" ]; then
    echo "Error: Archive not found: $ARCHIVE"
    if [ -d "$DIST_DIR" ]; then
        echo "Contents of $DIST_DIR:"
        ls -R "$DIST_DIR"
    fi
    exit 1
fi

TMP_DIR=$(mktemp -d)
# Ensure cleanup of BOTH temp directories
CLEANUP_HOME=""
cleanup() {
    echo "Cleaning up..."
    rm -rf "$TMP_DIR"
    if [ -n "$CLEANUP_HOME" ]; then
        rm -rf "$CLEANUP_HOME"
    fi
}
trap cleanup EXIT

echo "Extracting $ARCHIVE to $TMP_DIR..."
tar -xzf "$ARCHIVE" -C "$TMP_DIR"

BINARY="$TMP_DIR/whooper"

if [ ! -x "$BINARY" ]; then
    echo "Error: Binary not found or not executable in archive: $BINARY"
    echo "Contents of $TMP_DIR:"
    ls -lR "$TMP_DIR"
    exit 1
fi

echo "Verifying version..."
# Some environments might add a newline, so we trim it.
ACTUAL_VERSION=$("$BINARY" version | tr -d '\r\n')
if [ "$ACTUAL_VERSION" != "$VERSION" ]; then
    echo "Error: Version mismatch!"
    echo "Expected: '$VERSION'"
    echo "Actual:   '$ACTUAL_VERSION'"
    exit 1
fi
echo "Version check passed: $ACTUAL_VERSION"

echo "Running doctor --skip-api in a temporary home..."
CLEANUP_HOME=$(mktemp -d)
export WHOOPER_HOME="$CLEANUP_HOME"

# Run doctor --skip-api. This should succeed without network or real config.
"$BINARY" doctor --skip-api

echo "Smoke tests passed successfully for $ARCHIVE"
