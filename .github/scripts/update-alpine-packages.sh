#!/usr/bin/env bash
set -euo pipefail

# Called by Renovate postUpgradeTasks after an Alpine Docker image bump.
# Downloads the Alpine APK index for the new release and updates all
# repology-tracked package version ARGs in the given Dockerfile.
#
# Usage: update-alpine-packages.sh <packageFile>

PACKAGE_FILE="${1:?Usage: $0 <packageFile>}"

# Skip if the file has no repology-tracked Alpine packages
if ! grep -q 'datasource=repology depName=alpine_' "$PACKAGE_FILE" 2>/dev/null; then
    exit 0
fi

# Extract the Alpine version from the renovate comment lines in the file.
# After Renovate updates, these will already reflect the new version (e.g., alpine_3_23).
ALPINE_MAJOR_MINOR=$(grep -oP 'depName=alpine_\K\d+_\d+' "$PACKAGE_FILE" | head -1)
if [ -z "$ALPINE_MAJOR_MINOR" ]; then
    exit 0
fi

ALPINE_VERSION="v${ALPINE_MAJOR_MINOR//_/.}"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Download APK indices for main and community repos
for repo in main community; do
    mkdir -p "$TMPDIR/$repo"
    curl -sfL "https://dl-cdn.alpinelinux.org/alpine/${ALPINE_VERSION}/${repo}/x86_64/APKINDEX.tar.gz" \
        | tar xzf - -C "$TMPDIR/$repo" 2>/dev/null || true
done

# Look up a package version from the downloaded APK index
get_version() {
    local pkg="$1"
    local ver=""
    for repo in main community; do
        if [ -f "$TMPDIR/$repo/APKINDEX" ]; then
            ver=$(awk -v "pkg=$pkg" '
                /^P:/ { name = substr($0, 3) }
                /^V:/ { if (name == pkg) { print substr($0, 3); exit } }
            ' "$TMPDIR/$repo/APKINDEX")
            if [ -n "$ver" ]; then
                echo "$ver"
                return
            fi
        fi
    done
}

# Process each repology-tracked Alpine package in the Dockerfile
grep -n 'datasource=repology depName=alpine_' "$PACKAGE_FILE" | while IFS=: read -r line_num rest; do
    # Extract package name from depName=alpine_X_Y/pkgname
    pkg=$(echo "$rest" | sed -n 's/.*depName=alpine_[0-9]*_[0-9]*\/\([^ ]*\).*/\1/p')
    [ -z "$pkg" ] && continue

    new_ver=$(get_version "$pkg")
    [ -z "$new_ver" ] && continue

    # The ARG line should be the next line after the comment
    arg_line=$((line_num + 1))
    current=$(sed -n "${arg_line}p" "$PACKAGE_FILE")

    # Verify it's an ARG line before modifying
    if [[ "$current" =~ ^ARG\ [A-Za-z0-9_]+=.+ ]]; then
        old_ver="${current#*=}"
        if [ "$old_ver" != "$new_ver" ]; then
            sed -i "${arg_line}s|=.*|=${new_ver}|" "$PACKAGE_FILE"
        fi
    fi
done
