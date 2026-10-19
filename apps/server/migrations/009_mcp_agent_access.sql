CREATE TABLE mcp_oauth_clients (
  id text PRIMARY KEY,
  name text NOT NULL,
  redirect_uris jsonb NOT NULL,
  metadata_uri text,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE mcp_oauth_consents (
  owner_id text NOT NULL REFERENCES owners(id),
  client_id text NOT NULL REFERENCES mcp_oauth_clients(id),
  scopes text[] NOT NULL,
  granted_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (owner_id, client_id)
);

CREATE TABLE mcp_oauth_codes (
  hash text PRIMARY KEY,
  owner_id text NOT NULL REFERENCES owners(id),
  client_id text NOT NULL REFERENCES mcp_oauth_clients(id),
  redirect_uri text NOT NULL,
  resource text NOT NULL,
  scopes text[] NOT NULL,
  code_challenge text NOT NULL,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX mcp_oauth_codes_expiry ON mcp_oauth_codes(expires_at);

CREATE TABLE mcp_oauth_tokens (
  hash text PRIMARY KEY,
  owner_id text NOT NULL REFERENCES owners(id),
  client_id text NOT NULL REFERENCES mcp_oauth_clients(id),
  resource text NOT NULL,
  scopes text[] NOT NULL,
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX mcp_oauth_tokens_owner ON mcp_oauth_tokens(owner_id, client_id);
CREATE INDEX mcp_oauth_tokens_expiry ON mcp_oauth_tokens(expires_at);

CREATE TABLE mcp_upload_chunks (
  deployment_id text NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
  file_index integer NOT NULL CHECK (file_index >= 0),
  offset_bytes bigint NOT NULL CHECK (offset_bytes >= 0),
  size_bytes integer NOT NULL CHECK (size_bytes > 0),
  sha256 text NOT NULL CHECK (sha256 ~ '^[a-f0-9]{64}$'),
  object_key text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (deployment_id, file_index, offset_bytes)
);
CREATE INDEX mcp_upload_chunks_created ON mcp_upload_chunks(created_at);
