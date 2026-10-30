CREATE TABLE model_connections (
  id text PRIMARY KEY,
  owner_id text NOT NULL REFERENCES owners(id),
  name text NOT NULL CHECK (length(name) BETWEEN 1 AND 80),
  api_base text NOT NULL,
  model_id text NOT NULL,
  encrypted_key bytea NOT NULL CHECK (length(encrypted_key) > 0),
  key_version integer NOT NULL CHECK (key_version > 0),
  capabilities jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(capabilities) = 'object'),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, owner_id)
);
CREATE INDEX model_connections_owner ON model_connections(owner_id, created_at);

CREATE TABLE model_connection_grants (
  owner_id text NOT NULL REFERENCES owners(id),
  app_id text NOT NULL REFERENCES apps(id),
  connection_id text NOT NULL,
  granted_at timestamptz NOT NULL DEFAULT now(),
  revoked_at timestamptz,
  PRIMARY KEY (owner_id, app_id, connection_id),
  FOREIGN KEY (connection_id, owner_id) REFERENCES model_connections(id, owner_id) ON DELETE CASCADE
);
CREATE INDEX model_connection_grants_app ON model_connection_grants(app_id, owner_id) WHERE revoked_at IS NULL;

CREATE TABLE model_request_usage (
  id text PRIMARY KEY,
  owner_id text NOT NULL REFERENCES owners(id),
  app_id text NOT NULL REFERENCES apps(id),
  connection_id text NOT NULL,
  status text NOT NULL CHECK (status IN ('started', 'succeeded', 'failed', 'cancelled')),
  input_tokens bigint CHECK (input_tokens >= 0),
  output_tokens bigint CHECK (output_tokens >= 0),
  started_at timestamptz NOT NULL DEFAULT now(),
  finished_at timestamptz,
  CHECK ((status = 'started') = (finished_at IS NULL))
);
CREATE INDEX model_request_usage_owner_time ON model_request_usage(owner_id, started_at DESC);
CREATE INDEX model_request_usage_app_time ON model_request_usage(app_id, started_at DESC);
