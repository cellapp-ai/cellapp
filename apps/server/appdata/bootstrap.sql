-- Run as a PostgreSQL administrator connected to the dedicated app-data
-- database. Set passwords for the LOGIN roles outside this file.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'cellapp_appdata_provisioner') THEN
    CREATE ROLE cellapp_appdata_provisioner LOGIN CREATEROLE NOSUPERUSER NOCREATEDB NOREPLICATION NOBYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'cellapp_appdata_authenticator') THEN
    CREATE ROLE cellapp_appdata_authenticator LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'cellapp_appdata_anon') THEN
    CREATE ROLE cellapp_appdata_anon NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
  END IF;
END
$$;

DO $$
BEGIN
  EXECUTE format('REVOKE ALL ON DATABASE %I FROM PUBLIC', current_database());
  EXECUTE format('GRANT CONNECT, CREATE ON DATABASE %I TO cellapp_appdata_provisioner', current_database());
  EXECUTE format('GRANT CONNECT ON DATABASE %I TO cellapp_appdata_authenticator', current_database());
END
$$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
CREATE SCHEMA IF NOT EXISTS cellapp_internal AUTHORIZATION cellapp_appdata_provisioner;
ALTER SCHEMA cellapp_internal OWNER TO cellapp_appdata_provisioner;
CREATE TABLE IF NOT EXISTS cellapp_internal.app_versions (
  app_id text NOT NULL,
  version integer NOT NULL CHECK (version > 0),
  digest text NOT NULL,
  applied_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (app_id, version)
);
ALTER TABLE cellapp_internal.app_versions ADD COLUMN IF NOT EXISTS manifest jsonb;
ALTER TABLE cellapp_internal.app_versions OWNER TO cellapp_appdata_provisioner;
CREATE TABLE IF NOT EXISTS cellapp_internal.app_storage (
  app_id text PRIMARY KEY,
  bytes bigint NOT NULL DEFAULT 0 CHECK (bytes >= 0),
  limit_bytes bigint NOT NULL DEFAULT 50000000 CHECK (limit_bytes > 0)
);
ALTER TABLE cellapp_internal.app_storage OWNER TO cellapp_appdata_provisioner;
CREATE TABLE IF NOT EXISTS cellapp_internal.app_storage_tables (
  app_id text NOT NULL,
  table_name text NOT NULL,
  bytes bigint NOT NULL DEFAULT 0 CHECK (bytes >= 0),
  PRIMARY KEY (app_id, table_name)
);
ALTER TABLE cellapp_internal.app_storage_tables OWNER TO cellapp_appdata_provisioner;
REVOKE ALL ON SCHEMA cellapp_internal FROM PUBLIC;
REVOKE ALL ON ALL TABLES IN SCHEMA cellapp_internal FROM PUBLIC;
GRANT USAGE ON SCHEMA cellapp_internal TO cellapp_appdata_authenticator;
CREATE SCHEMA IF NOT EXISTS cellapp_api AUTHORIZATION cellapp_appdata_provisioner;
REVOKE ALL ON SCHEMA cellapp_api FROM PUBLIC;
GRANT USAGE ON SCHEMA cellapp_api TO cellapp_appdata_authenticator, cellapp_appdata_anon;

-- PostgREST reloads this list when the provisioner sends NOTIFY pgrst.
-- The empty cellapp_api schema keeps startup valid before the first app exists.
CREATE OR REPLACE FUNCTION cellapp_internal.pre_config() RETURNS void
LANGUAGE sql SECURITY DEFINER SET search_path = pg_catalog AS $$
  SELECT set_config('pgrst.db_schemas',
    'cellapp_api' || COALESCE(',' || string_agg(nspname, ',' ORDER BY nspname), ''), true)
  FROM pg_namespace
  WHERE nspname ~ '^app_[a-f0-9]{32}$'
$$;
ALTER FUNCTION cellapp_internal.pre_config() OWNER TO cellapp_appdata_provisioner;
REVOKE ALL ON FUNCTION cellapp_internal.pre_config() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION cellapp_internal.pre_config() TO cellapp_appdata_authenticator;
GRANT cellapp_appdata_anon TO cellapp_appdata_authenticator;

CREATE OR REPLACE FUNCTION cellapp_internal.pre_request() RETURNS void
LANGUAGE sql SET search_path = pg_catalog AS $$
  SELECT set_config('statement_timeout', '5000', true),
         set_config('lock_timeout', '1000', true)
$$;
ALTER FUNCTION cellapp_internal.pre_request() OWNER TO cellapp_appdata_provisioner;
REVOKE ALL ON FUNCTION cellapp_internal.pre_request() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION cellapp_internal.pre_request() TO cellapp_appdata_anon;

CREATE OR REPLACE FUNCTION cellapp_internal.account_storage() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $$
DECLARE change_bytes bigint := 0;
BEGIN
  IF TG_OP <> 'INSERT' THEN
    change_bytes := change_bytes - octet_length(to_jsonb(OLD)::text) - 128;
  END IF;
  IF TG_OP <> 'DELETE' THEN
    change_bytes := change_bytes + octet_length(to_jsonb(NEW)::text) + 128;
  END IF;
  UPDATE cellapp_internal.app_storage
    SET bytes = bytes + change_bytes
    WHERE app_id = substring(TG_TABLE_SCHEMA from 5)
      AND bytes + change_bytes BETWEEN 0 AND limit_bytes;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'application storage limit exceeded' USING ERRCODE = 'P0001';
  END IF;
  UPDATE cellapp_internal.app_storage_tables
    SET bytes = bytes + change_bytes
    WHERE app_id = substring(TG_TABLE_SCHEMA from 5)
      AND table_name = TG_TABLE_NAME
      AND bytes + change_bytes >= 0;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'application storage accounting unavailable' USING ERRCODE = 'P0001';
  END IF;
  IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
  RETURN NEW;
END
$$;
ALTER FUNCTION cellapp_internal.account_storage() OWNER TO cellapp_appdata_provisioner;
REVOKE ALL ON FUNCTION cellapp_internal.account_storage() FROM PUBLIC;

-- App DDL roles own their schemas but cannot introduce executable database
-- objects or delegate their privileges, even if an operator sets the role.
CREATE OR REPLACE FUNCTION cellapp_internal.guard_app_ddl() RETURNS event_trigger
LANGUAGE plpgsql AS $$
BEGIN
  IF current_user ~ '^app_ddl_[a-f0-9]{32}$' AND TG_TAG NOT IN (
    'CREATE TABLE', 'ALTER TABLE', 'DROP TABLE',
    'CREATE INDEX', 'ALTER INDEX', 'DROP INDEX',
    'CREATE POLICY', 'ALTER POLICY', 'DROP POLICY'
  ) THEN
    RAISE EXCEPTION 'operation is not allowed for application DDL role' USING ERRCODE = '42501';
  END IF;
END
$$;
REVOKE ALL ON FUNCTION cellapp_internal.guard_app_ddl() FROM PUBLIC;
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_event_trigger WHERE evtname = 'cellapp_guard_app_ddl') THEN
    CREATE EVENT TRIGGER cellapp_guard_app_ddl ON ddl_command_start
      EXECUTE FUNCTION cellapp_internal.guard_app_ddl();
  END IF;
END
$$;
