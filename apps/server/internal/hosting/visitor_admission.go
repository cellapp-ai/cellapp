package hosting

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// admitVisitor runs after identity and access checks, in the caller's transaction.
// Locking the month row serializes first visits from different users at the cap.
func admitVisitor(ctx context.Context, tx pgx.Tx, appID, platformUser string, owner bool, limit int64, now time.Time) error {
	if owner {
		return nil
	}
	now = now.UTC()
	period := now.Format("2006-01")
	if _, err := tx.Exec(ctx, `INSERT INTO app_monthly_visitors(app_id,period) VALUES($1,$2) ON CONFLICT DO NOTHING`, appID, period); err != nil {
		return err
	}
	var admitted int64
	if err := tx.QueryRow(ctx, `SELECT admitted FROM app_monthly_visitors WHERE app_id=$1 AND period=$2 FOR UPDATE`, appID, period).Scan(&admitted); err != nil {
		return err
	}
	var existing bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app_visitor_admissions WHERE app_id=$1 AND period=$2 AND platform_user_id=$3)`, appID, period, platformUser).Scan(&existing); err != nil {
		return err
	}
	if existing {
		return nil
	}
	if admitted >= limit {
		reset := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		return &APIError{429, "visitor_limit", "Monthly visitor limit reached; try after reset", map[string]string{"resetAt": reset.Format(time.RFC3339)}}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app_visitor_admissions(app_id,period,platform_user_id) VALUES($1,$2,$3)`, appID, period, platformUser); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE app_monthly_visitors SET admitted=admitted+1 WHERE app_id=$1 AND period=$2`, appID, period)
	return err
}
