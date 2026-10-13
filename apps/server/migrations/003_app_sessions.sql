CREATE TABLE app_handoff_tickets (
  hash text PRIMARY KEY,
  app_id text NOT NULL REFERENCES apps(id),
  browser_hash text NOT NULL,
  return_path text NOT NULL,
  expires_at timestamptz NOT NULL
);
CREATE INDEX app_handoff_expiry ON app_handoff_tickets(expires_at);

CREATE TABLE app_browser_sessions (
  hash text PRIMARY KEY,
  app_id text NOT NULL REFERENCES apps(id),
  browser_hash text NOT NULL,
  platform_user_id text NOT NULL REFERENCES owners(id),
  expires_at timestamptz NOT NULL
);
CREATE INDEX app_browser_sessions_expiry ON app_browser_sessions(expires_at);
