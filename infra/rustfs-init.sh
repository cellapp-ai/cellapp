#!/bin/sh
set -eu

: "${S3_ENDPOINT:?S3_ENDPOINT is required}"
: "${S3_BUCKET:?S3_BUCKET is required}"
: "${S3_ACCESS_KEY_ID:?S3_ACCESS_KEY_ID is required}"
: "${S3_SECRET_ACCESS_KEY:?S3_SECRET_ACCESS_KEY is required}"
S3_REGION=${S3_REGION:-us-east-1}
S3_INIT_MAX_ATTEMPTS=${S3_INIT_MAX_ATTEMPTS:-20}
case "$S3_INIT_MAX_ATTEMPTS" in
  ''|*[!0-9]*) echo "S3_INIT_MAX_ATTEMPTS must be a positive integer." >&2; exit 1 ;;
esac
if [ "$S3_INIT_MAX_ATTEMPTS" -lt 1 ]; then
  echo "S3_INIT_MAX_ATTEMPTS must be a positive integer." >&2
  exit 1
fi

set -- alias set --bucket-lookup path --region "$S3_REGION"
if [ -n "${S3_CA_BUNDLE:-}" ]; then
  set -- "$@" --ca-bundle "$S3_CA_BUNDLE"
fi
attempt=0
until rc "$@" local "$S3_ENDPOINT" "$S3_ACCESS_KEY_ID" "$S3_SECRET_ACCESS_KEY" >/tmp/cellapp-alias-output 2>&1 &&
  rc ready --timeout 5 local >/tmp/cellapp-ready-output 2>&1; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge "$S3_INIT_MAX_ATTEMPTS" ]; then
    echo "RustFS connection/readiness failed: check endpoint, TLS and credentials." >&2
    cat /tmp/cellapp-alias-output /tmp/cellapp-ready-output 2>/dev/null >&2 || true
    exit 1
  fi
  sleep 2
done

rc bucket create --ignore-existing "local/$S3_BUCKET"
rc bucket anonymous set private "local/$S3_BUCKET"
policy=$(rc bucket anonymous get "local/$S3_BUCKET")
case "$policy" in
  *"'private'"*) ;;
  *) echo "RustFS private bucket policy check failed: $policy" >&2; exit 1 ;;
esac

probe=".cellapp-private-probe/$$"
trap 'rc object remove "local/$S3_BUCKET/$probe" >/dev/null 2>&1 || true' EXIT
printf 'cellapp-private-probe' | rc pipe "local/$S3_BUCKET/$probe" >/dev/null
set -- alias set --anonymous --bucket-lookup path --region "$S3_REGION"
if [ -n "${S3_CA_BUNDLE:-}" ]; then
  set -- "$@" --ca-bundle "$S3_CA_BUNDLE"
fi
rc "$@" anonymous "$S3_ENDPOINT" >/dev/null
if rc object show "anonymous/$S3_BUCKET/$probe" >/dev/null 2>/tmp/cellapp-anonymous-error; then
  echo "RustFS anonymous GET exposed a private object." >&2
  exit 1
fi
if ! grep -Eiq 'AccessDenied|403|Forbidden' /tmp/cellapp-anonymous-error; then
  echo "RustFS anonymous GET failed without an access denial:" >&2
  cat /tmp/cellapp-anonymous-error >&2
  exit 1
fi
echo "Private RustFS bucket initialized and anonymous GET denied."
