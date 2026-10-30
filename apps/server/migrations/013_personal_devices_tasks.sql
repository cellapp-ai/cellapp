CREATE TABLE personal_devices (
  id text PRIMARY KEY,
  owner_id text NOT NULL REFERENCES owners(id),
  name text NOT NULL CHECK (length(name) BETWEEN 1 AND 80),
  credential_hash text NOT NULL UNIQUE,
  capabilities jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(capabilities) = 'object'),
  paired_at timestamptz NOT NULL DEFAULT now(),
  last_seen_at timestamptz,
  revoked_at timestamptz,
  UNIQUE (owner_id, id)
);
CREATE INDEX personal_devices_owner ON personal_devices(owner_id, paired_at DESC);

CREATE TABLE device_projects (
  id text PRIMARY KEY,
  owner_id text NOT NULL,
  device_id text NOT NULL,
  name text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
  permissions jsonb NOT NULL CHECK (jsonb_typeof(permissions) = 'object'),
  created_at timestamptz NOT NULL DEFAULT now(),
  revoked_at timestamptz,
  FOREIGN KEY (owner_id, device_id) REFERENCES personal_devices(owner_id, id) ON DELETE CASCADE,
  UNIQUE (owner_id, device_id, id)
);
CREATE INDEX device_projects_owner ON device_projects(owner_id, device_id, created_at DESC);

CREATE TABLE remote_agent_tasks (
  id text PRIMARY KEY,
  owner_id text NOT NULL,
  device_id text NOT NULL,
  project_id text NOT NULL,
  kind text NOT NULL CHECK (kind IN ('develop', 'run_agent', 'install_agent')),
  request jsonb NOT NULL CHECK (jsonb_typeof(request) = 'object'),
  state text NOT NULL DEFAULT 'queued' CHECK (state IN ('queued', 'claimed', 'running', 'cancel_requested', 'cancelled', 'succeeded', 'failed')),
  sequence bigint NOT NULL DEFAULT 0 CHECK (sequence >= 0),
  progress jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(progress) = 'object'),
  result jsonb CHECK (result IS NULL OR jsonb_typeof(result) = 'object'),
  created_at timestamptz NOT NULL DEFAULT now(),
  claimed_at timestamptz,
  finished_at timestamptz,
  CHECK ((state IN ('cancelled', 'succeeded', 'failed')) = (finished_at IS NOT NULL)),
  CHECK (state <> 'queued' OR (claimed_at IS NULL AND sequence = 0)),
  FOREIGN KEY (owner_id, device_id, project_id) REFERENCES device_projects(owner_id, device_id, id)
);
CREATE INDEX remote_agent_tasks_owner ON remote_agent_tasks(owner_id, created_at DESC);
CREATE INDEX remote_agent_tasks_device_queue ON remote_agent_tasks(device_id, created_at) WHERE state = 'queued';
