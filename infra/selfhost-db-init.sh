#!/bin/sh
set -eu
export PGPASSWORD="$POSTGRES_PASSWORD"
export PGHOST=postgres PGUSER=cellapp_admin

psql -d postgres -v ON_ERROR_STOP=1 \
  -v control_password="$CONTROL_DB_PASSWORD" \
  -v provisioner_password="$APP_DATA_DB_PASSWORD" \
  -v authenticator_password="$POSTGREST_DB_PASSWORD" <<'SQL'
SELECT 'CREATE ROLE cellapp_control LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS'
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'cellapp_control') \gexec
SELECT 'CREATE DATABASE cellapp OWNER cellapp_control'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'cellapp') \gexec
SELECT 'CREATE DATABASE cellapp_data'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'cellapp_data') \gexec
SELECT format('ALTER ROLE cellapp_control PASSWORD %L', :'control_password') \gexec
REVOKE CONNECT ON DATABASE cellapp FROM PUBLIC;
GRANT CONNECT ON DATABASE cellapp TO cellapp_control;
SQL

# The bootstrap is idempotent and creates the app-data roles without passwords.
psql -d cellapp_data -v ON_ERROR_STOP=1 -f /bootstrap-appdata.sql >/dev/null
psql -d cellapp_data -v ON_ERROR_STOP=1 \
  -v provisioner_password="$APP_DATA_DB_PASSWORD" \
  -v authenticator_password="$POSTGREST_DB_PASSWORD" <<'SQL'
SELECT format('ALTER ROLE cellapp_appdata_provisioner PASSWORD %L', :'provisioner_password') \gexec
SELECT format('ALTER ROLE cellapp_appdata_authenticator PASSWORD %L', :'authenticator_password') \gexec
SQL
echo 'Application databases initialized.'
