#!/bin/sh
set -e

# Fix data directory permissions when running as root.
# Docker named volumes / host bind-mounts may be owned by root,
# preventing the non-root sub2api user from writing files.
if [ "$(id -u)" = "0" ]; then
    payment_private_dir="${PAYMENT_PRIVATE_STORAGE_DIR:-/data/payment-private}"
    mkdir -p /app/data "$payment_private_dir"
    payment_private_dir="$(readlink -f "$payment_private_dir")"
    if [ -z "$payment_private_dir" ] || [ "$payment_private_dir" = "/" ]; then
        echo "PAYMENT_PRIVATE_STORAGE_DIR must resolve to a dedicated directory" >&2
        exit 1
    fi
    # Use || true to avoid failure on read-only mounted files (e.g. config.yaml:ro)
    chown -R sub2api:sub2api /app/data 2>/dev/null || true
    chown -R sub2api:sub2api "$payment_private_dir" 2>/dev/null || true
    chmod 700 "$payment_private_dir" 2>/dev/null || true
    # Re-invoke this script as sub2api so the flag-detection below
    # also runs under the correct user.
    exec su-exec sub2api "$0" "$@"
fi

# Compatibility: if the first arg looks like a flag (e.g. --help),
# prepend the default binary so it behaves the same as the old
# ENTRYPOINT ["/app/sub2api"] style.
if [ "${1#-}" != "$1" ]; then
    set -- /app/sub2api "$@"
fi

exec "$@"
