ALTER TABLE apps ADD COLUMN IF NOT EXISTS data_provider text;
ALTER TABLE apps ADD COLUMN IF NOT EXISTS data_url text;
ALTER TABLE apps ADD COLUMN IF NOT EXISTS data_anon_key text;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint c
    JOIN pg_class t ON c.conrelid = t.oid
    JOIN pg_namespace n ON t.relnamespace = n.oid
    WHERE c.conname = 'apps_data_provider_check' AND n.nspname = current_schema()
  ) THEN
    ALTER TABLE apps ADD CONSTRAINT apps_data_provider_check
      CHECK (data_provider IS NULL OR data_provider = 'supabase');
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint c
    JOIN pg_class t ON c.conrelid = t.oid
    JOIN pg_namespace n ON t.relnamespace = n.oid
    WHERE c.conname = 'apps_data_complete_check' AND n.nspname = current_schema()
  ) THEN
    ALTER TABLE apps ADD CONSTRAINT apps_data_complete_check CHECK (
      (data_provider IS NULL AND data_url IS NULL AND data_anon_key IS NULL)
      OR (data_provider IS NOT NULL AND data_url IS NOT NULL AND data_anon_key IS NOT NULL)
    );
  END IF;
END $$;
