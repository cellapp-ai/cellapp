#!/bin/sh
set -eu

: "${S3_ENDPOINT:?S3_ENDPOINT is required}"
: "${S3_BUCKET:?S3_BUCKET is required}"
: "${S3_ACCESS_KEY_ID:?S3_ACCESS_KEY_ID is required}"
: "${S3_SECRET_ACCESS_KEY:?S3_SECRET_ACCESS_KEY is required}"

# Bound retries and each request; never report success before the private bucket is ready.
attempt=0
until mc --conn-read-deadline 5s --conn-write-deadline 5s alias set local "$S3_ENDPOINT" "$S3_ACCESS_KEY_ID" "$S3_SECRET_ACCESS_KEY" >/dev/null 2>&1 &&
  mc --conn-read-deadline 5s --conn-write-deadline 5s mb --ignore-existing "local/$S3_BUCKET" >/dev/null 2>&1; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 20 ]; then
    echo "MinIO initialization failed: check endpoint, credentials and container readiness." >&2
    exit 1
  fi
  sleep 2
done

mc --conn-read-deadline 5s --conn-write-deadline 5s anonymous set none "local/$S3_BUCKET"
echo "Private local S3 bucket initialized."
