ALTER TABLE owners ADD COLUMN github_login text NOT NULL DEFAULT ''
  CHECK (char_length(github_login) <= 39);
