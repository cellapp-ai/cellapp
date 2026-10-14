CREATE TABLE app_monthly_visitors (
  app_id text NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
  period text NOT NULL CHECK (period ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
  admitted bigint NOT NULL DEFAULT 0 CHECK (admitted >= 0),
  PRIMARY KEY (app_id, period)
);

CREATE TABLE app_visitor_admissions (
  app_id text NOT NULL,
  period text NOT NULL,
  platform_user_id text NOT NULL REFERENCES owners(id),
  admitted_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (app_id, period, platform_user_id),
  FOREIGN KEY (app_id, period) REFERENCES app_monthly_visitors(app_id, period) ON DELETE CASCADE
);
