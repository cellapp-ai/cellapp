#!/bin/sh
set -eu

# Local development only. The init hook runs on a fresh PostgreSQL volume;
# operators can invoke it again for an existing local volume.
if [ "$(psql -U "$POSTGRES_USER" -d postgres -Atqc "SELECT 1 FROM pg_database WHERE datname='cellapp_data'")" != "1" ]; then
  psql -U "$POSTGRES_USER" -d postgres -v ON_ERROR_STOP=1 -c 'CREATE DATABASE cellapp_data'
fi
psql -U "$POSTGRES_USER" -d cellapp_data -v ON_ERROR_STOP=1 -f /bootstrap-appdata.sql
psql -U "$POSTGRES_USER" -d cellapp_data -v ON_ERROR_STOP=1 -c "ALTER ROLE cellapp_appdata_provisioner PASSWORD 'development'"
psql -U "$POSTGRES_USER" -d cellapp_data -v ON_ERROR_STOP=1 -c "ALTER ROLE cellapp_appdata_authenticator PASSWORD 'development'"
