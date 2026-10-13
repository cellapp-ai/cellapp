-- Host labels stay assigned to the original application after a rename or deletion.
ALTER TABLE apps ADD COLUMN host_label text
  CHECK (host_label IS NULL OR host_label ~ '^[a-z][a-z0-9-]{0,62}$');
CREATE UNIQUE INDEX apps_host_label_unique ON apps(host_label) WHERE host_label IS NOT NULL;
CREATE TABLE app_hostnames (
  label text PRIMARY KEY CHECK (label ~ '^[a-z][a-z0-9-]{0,62}$'),
  app_id text NOT NULL REFERENCES apps(id)
);
CREATE TABLE app_alias_history (
  namespace text NOT NULL,
  slug text NOT NULL,
  app_id text NOT NULL REFERENCES apps(id),
  PRIMARY KEY(namespace,slug)
);
-- Existing long or ambiguous aliases retain their ID host until the owner sets shorter slugs.
WITH candidates AS (
  SELECT a.id, a.slug || '-' || n.handle AS label,
    row_number() OVER (PARTITION BY a.slug || '-' || n.handle ORDER BY a.id) AS rank
  FROM apps a JOIN app_namespaces n ON n.id=a.namespace_id
  WHERE length(a.slug || '-' || n.handle) <= 63
    AND length(n.handle) >= 3
    AND right(n.handle,1) <> '-'
    AND right(a.slug,1) <> '-'
    AND n.handle NOT IN ('accounts','account','admin','api','app','apps','assets','auth','billing','cdn','cloud','console','dashboard','docs','files','help','identity','login','logout','metrics','oauth','static','status','support','system','www')
    AND a.slug NOT IN ('accounts','account','admin','api','app','apps','assets','auth','billing','cdn','cloud','console','dashboard','docs','files','help','identity','login','logout','metrics','oauth','static','status','support','system','www')
)
UPDATE apps a SET host_label=c.label FROM candidates c WHERE a.id=c.id AND c.rank=1;
INSERT INTO app_hostnames(label,app_id)
  SELECT host_label,id FROM apps WHERE host_label IS NOT NULL;
