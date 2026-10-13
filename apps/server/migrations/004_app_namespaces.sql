-- Public aliases are separate from the immutable application ID and host.
CREATE TABLE app_namespaces (
  id text PRIMARY KEY CHECK (id ~ '^[a-f0-9]{32}$'),
  owner_id text NOT NULL UNIQUE REFERENCES owners(id),
  handle text NOT NULL UNIQUE CHECK (handle ~ '^[a-z][a-z0-9-]{0,62}$'),
  created_at timestamptz NOT NULL DEFAULT now()
);
WITH generated AS (
  SELECT id AS owner_id, replace(gen_random_uuid()::text, '-', '') AS namespace_id FROM owners
)
INSERT INTO app_namespaces(id,owner_id,handle)
SELECT namespace_id,owner_id,'n-' || namespace_id FROM generated;
ALTER TABLE apps ADD COLUMN namespace_id text REFERENCES app_namespaces(id);
ALTER TABLE apps ADD COLUMN slug text;
UPDATE apps SET namespace_id=n.id, slug='app-' || apps.id
FROM app_namespaces n WHERE n.owner_id=apps.owner_id;
ALTER TABLE apps ALTER COLUMN namespace_id SET NOT NULL;
ALTER TABLE apps ALTER COLUMN slug SET NOT NULL;
ALTER TABLE apps ADD CONSTRAINT apps_slug_format CHECK (slug ~ '^[a-z][a-z0-9-]{0,62}$');
-- Keep tombstoned names occupied so old links never resolve to another app.
CREATE UNIQUE INDEX apps_namespace_slug ON apps(namespace_id,slug);
