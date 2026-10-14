ALTER TABLE owners ADD COLUMN display_name text NOT NULL DEFAULT ''
  CHECK (char_length(display_name) <= 80 AND display_name = btrim(display_name));
