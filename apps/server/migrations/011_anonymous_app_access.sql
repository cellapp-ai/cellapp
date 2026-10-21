ALTER TABLE apps
  ADD COLUMN anonymous_pages boolean NOT NULL DEFAULT false,
  ADD COLUMN anonymous_read boolean NOT NULL DEFAULT false,
  ADD COLUMN anonymous_write boolean NOT NULL DEFAULT false;

ALTER TABLE apps
  ADD CONSTRAINT apps_anonymous_access_requires_public
    CHECK (visibility = 'public' OR NOT (anonymous_pages OR anonymous_read OR anonymous_write)),
  ADD CONSTRAINT apps_anonymous_data_requires_pages
    CHECK (anonymous_pages OR NOT (anonymous_read OR anonymous_write));
