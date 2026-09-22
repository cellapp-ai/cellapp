CREATE TABLE IF NOT EXISTS owners (
  id text PRIMARY KEY, issuer text NOT NULL, subject text NOT NULL,
  UNIQUE(issuer, subject)
);
CREATE TABLE IF NOT EXISTS browser_sessions (
  hash text PRIMARY KEY, owner_id text REFERENCES owners(id),
  data jsonb NOT NULL DEFAULT '{}', expires_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS device_authorizations (
  hash text PRIMARY KEY, user_code text NOT NULL UNIQUE,
  owner_id text REFERENCES owners(id), status text NOT NULL DEFAULT 'pending'
    CHECK(status IN ('pending','approved','denied','claimed')),
  expires_at timestamptz NOT NULL, last_poll timestamptz
);
CREATE TABLE IF NOT EXISTS credentials (
  id text PRIMARY KEY, hash text NOT NULL UNIQUE, owner_id text NOT NULL REFERENCES owners(id),
  expires_at timestamptz NOT NULL, revoked boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS apps (
  id text PRIMARY KEY, owner_id text NOT NULL REFERENCES owners(id), name text NOT NULL,
  key_hash text NOT NULL, generation integer NOT NULL DEFAULT 1,
  active_deployment text, deleted boolean NOT NULL DEFAULT false,
  suspended boolean NOT NULL DEFAULT false, create_key text NOT NULL, create_digest text NOT NULL,
  UNIQUE(owner_id, create_key)
);
CREATE TABLE IF NOT EXISTS deployments (
  id text PRIMARY KEY, app_id text NOT NULL REFERENCES apps(id), idempotency_key text NOT NULL,
  digest text NOT NULL, manifest jsonb NOT NULL, uploaded jsonb NOT NULL DEFAULT '[]',
  base_version text, spa boolean NOT NULL DEFAULT false,
  status text NOT NULL DEFAULT 'uploading' CHECK(status IN ('uploading','published','cleaned')),
  bytes bigint NOT NULL CHECK(bytes >= 0), created_at timestamptz NOT NULL DEFAULT now(),
  published_at timestamptz, UNIQUE(app_id, idempotency_key)
);
CREATE TABLE IF NOT EXISTS access_sessions (
  hash text PRIMARY KEY, app_id text NOT NULL REFERENCES apps(id),
  generation integer NOT NULL, expires_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS usage (
  app_id text NOT NULL REFERENCES apps(id), period text NOT NULL,
  bytes bigint NOT NULL DEFAULT 0 CHECK(bytes >= 0), PRIMARY KEY(app_id, period)
);
CREATE TABLE IF NOT EXISTS rate_limits (
  key text PRIMARY KEY, count integer NOT NULL, expires_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS deployments_app ON deployments(app_id);
CREATE INDEX IF NOT EXISTS apps_owner ON apps(owner_id);
CREATE INDEX IF NOT EXISTS sessions_expiry ON access_sessions(expires_at);
