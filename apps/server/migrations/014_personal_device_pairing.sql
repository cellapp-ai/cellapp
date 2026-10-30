CREATE TABLE personal_device_authorizations (
  secret_hash text PRIMARY KEY,
  user_code text NOT NULL UNIQUE,
  name text NOT NULL CHECK (length(name) BETWEEN 1 AND 80),
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'denied', 'claimed')),
  owner_id text REFERENCES owners(id),
  expires_at timestamptz NOT NULL,
  last_poll_at timestamptz,
  CHECK ((status IN ('approved', 'claimed')) = (owner_id IS NOT NULL))
);
CREATE INDEX personal_device_authorizations_expiry ON personal_device_authorizations(expires_at);
