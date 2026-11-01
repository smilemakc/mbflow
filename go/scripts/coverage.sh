#!/usr/bin/env bash
# Measures test coverage of the backend module and enforces a minimum.
#
# Usage: scripts/coverage.sh [min-percent] [go test flags...]
#   COVER_EXEC=<wrapper>  run test binaries through a wrapper (go test -exec),
#                         e.g. to run embedded PostgreSQL as a non-root user.
#
# Generated code (protobuf, swagger), examples and test helpers are excluded
# from the measurement: they are not product code.
set -euo pipefail

cd "$(dirname "$0")/.."

MIN="${1:-90}"
shift || true
OUT="${COVER_OUT:-coverage.out}"

PKGS=$(go list ./... | grep -v -E '/examples/|/api/proto/|/docs/swagger|/testutil$')

exec_flag=()
if [ -n "${COVER_EXEC:-}" ]; then
	exec_flag=(-exec "$COVER_EXEC")
fi

go test -short -count=1 "${exec_flag[@]}" -covermode=atomic -coverprofile="$OUT.raw" "$@" $PKGS

# Drop generated files that may still appear in the profile.
grep -v -E '\.pb\.go:|\.pb\.gw\.go:|_grpc\.pb\.go:' "$OUT.raw" >"$OUT"
rm -f "$OUT.raw"

total=$(go tool cover -func="$OUT" | awk '/^total:/ {sub("%", "", $3); print $3}')
echo "total coverage: ${total}% (minimum ${MIN}%)"

if awk -v t="$total" -v m="$MIN" 'BEGIN { exit !(t + 0 < m + 0) }'; then
	echo "coverage ${total}% is below the required ${MIN}%" >&2
	exit 1
fi
