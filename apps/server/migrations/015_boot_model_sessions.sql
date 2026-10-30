CREATE TABLE boot_model_sessions (
  token_hash text PRIMARY KEY,
  owner_id text NOT NULL,
  device_id text NOT NULL,
  connection_id text NOT NULL,
  model_id text NOT NULL CHECK (length(model_id) BETWEEN 1 AND 256),
  expires_at timestamptz NOT NULL,
  calls integer NOT NULL DEFAULT 0 CHECK (calls BETWEEN 0 AND 6),
  active boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (owner_id, device_id) REFERENCES personal_devices(owner_id, id) ON DELETE CASCADE,
  FOREIGN KEY (connection_id, owner_id) REFERENCES model_connections(id, owner_id) ON DELETE CASCADE
);
CREATE INDEX boot_model_sessions_device ON boot_model_sessions(owner_id, device_id, expires_at);
