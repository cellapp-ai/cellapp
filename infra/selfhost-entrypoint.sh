#!/bin/sh
set -eu
: "${SELFHOST_CA_CERT:?internal CA certificate is required}"
test -r "$SELFHOST_CA_CERT"
cat /etc/ssl/certs/ca-certificates.crt "$SELFHOST_CA_CERT" > /tmp/cellapp-ca-bundle.crt
export SSL_CERT_FILE=/tmp/cellapp-ca-bundle.crt
exec /usr/local/bin/cloudrun-entrypoint
