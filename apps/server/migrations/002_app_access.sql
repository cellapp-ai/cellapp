ALTER TABLE apps
  ADD COLUMN visibility text NOT NULL DEFAULT 'restricted'
    CHECK (visibility IN ('restricted', 'public')),
  ADD COLUMN data_state text NOT NULL DEFAULT 'pending'
    CHECK (data_state IN ('pending', 'ready', 'error', 'cleanup_pending', 'cleaned')),
  ADD COLUMN data_error_code text,
  ADD COLUMN data_updated_at timestamptz NOT NULL DEFAULT now();

CREATE TABLE app_users (
  id text PRIMARY KEY,
  app_id text NOT NULL REFERENCES apps(id),
  platform_user_id text NOT NULL REFERENCES owners(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (app_id, platform_user_id),
  UNIQUE (app_id, id)
);
CREATE INDEX app_users_platform_user ON app_users(platform_user_id);

CREATE TABLE app_grants (
  app_id text NOT NULL REFERENCES apps(id),
  platform_user_id text NOT NULL REFERENCES owners(id),
  source text NOT NULL CHECK (source IN ('key', 'request', 'owner', 'purchase')),
  granted_at timestamptz NOT NULL DEFAULT now(),
  revoked_at timestamptz,
  PRIMARY KEY (app_id, platform_user_id)
);
CREATE INDEX app_grants_platform_user ON app_grants(platform_user_id);

CREATE TABLE app_access_requests (
  app_id text NOT NULL REFERENCES apps(id),
  platform_user_id text NOT NULL REFERENCES owners(id),
  status text NOT NULL CHECK (status IN ('pending', 'approved', 'denied')),
  requested_at timestamptz NOT NULL DEFAULT now(),
  decided_at timestamptz,
  PRIMARY KEY (app_id, platform_user_id)
);
CREATE INDEX app_access_requests_status ON app_access_requests(app_id, status);
