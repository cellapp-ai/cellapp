package hosting

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func slugConflict(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return fail(http.StatusConflict, "slug_taken", "This address is already in use")
	}
	return err
}

func claimHost(r *http.Request, tx pgx.Tx, label, appID string) error {
	var assigned string
	err := tx.QueryRow(r.Context(), `SELECT app_id FROM app_hostnames WHERE label=$1`, label).Scan(&assigned)
	if err == nil {
		if assigned != appID {
			return fail(http.StatusConflict, "slug_taken", "This address is already in use")
		}
		return nil
	}
	if err != pgx.ErrNoRows {
		return err
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO app_hostnames(label,app_id) VALUES($1,$2)`, label, appID)
	return slugConflict(err)
}

func (s *Server) webUpdateAppSlug(w http.ResponseWriter, r *http.Request, owner string) error {
	var input struct {
		Slug *string `json:"slug"`
	}
	if err := decode(w, r, &input); err != nil {
		return err
	}
	if input.Slug == nil {
		return fail(400, "invalid_slug", "Enter an application slug")
	}
	var app App
	err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		if err := lockOwner(r.Context(), tx, owner); err != nil {
			return err
		}
		var err error
		app, err = s.app(r.Context(), tx, r.PathValue("app"), owner)
		if err != nil {
			return err
		}
		if !validAppSlug(*input.Slug, app.Namespace) {
			return fail(400, "invalid_slug", "Application slug is reserved, invalid or too long for the user slug")
		}
		if *input.Slug == app.Slug {
			return nil
		}
		var occupied bool
		err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM apps a JOIN app_namespaces n ON n.id=a.namespace_id
    WHERE n.handle=$1 AND a.slug=$2 AND a.id<>$3) OR EXISTS(SELECT 1 FROM app_alias_history WHERE namespace=$1 AND slug=$2 AND app_id<>$3)`, app.Namespace, *input.Slug, app.ID).Scan(&occupied)
		if err != nil {
			return err
		}
		if occupied {
			return fail(409, "slug_taken", "This application slug is already in use")
		}
		newLabel := *input.Slug + "-" + app.Namespace
		if err = claimHost(r, tx, newLabel, app.ID); err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO app_alias_history(namespace,slug,app_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, app.Namespace, app.Slug, app.ID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `UPDATE apps SET slug=$1,host_label=$2 WHERE id=$3`, *input.Slug, newLabel, app.ID)
		if err != nil {
			return slugConflict(err)
		}
		app.Slug, app.HostLabel = *input.Slug, newLabel
		app.URL, app.AliasURL = s.appURL(app.ID, newLabel), s.appAliasURL(app.Namespace, app.Slug)
		return nil
	})
	if err != nil {
		return err
	}
	jsonResponse(w, 200, app)
	return nil
}

func (s *Server) webUpdateUserSlug(w http.ResponseWriter, r *http.Request, owner string) error {
	var input struct {
		UserSlug *string `json:"userSlug"`
	}
	if err := decode(w, r, &input); err != nil {
		return err
	}
	if input.UserSlug == nil || !validUserSlug(*input.UserSlug) {
		return fail(400, "invalid_user_slug", "Use 3–61 lowercase letters, digits or hyphens; reserved names are unavailable")
	}
	err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		if err := lockOwner(r.Context(), tx, owner); err != nil {
			return err
		}
		var namespaceID, old string
		err := tx.QueryRow(r.Context(), `SELECT id,handle FROM app_namespaces WHERE owner_id=$1 FOR UPDATE`, owner).Scan(&namespaceID, &old)
		if err == pgx.ErrNoRows {
			_, err = tx.Exec(r.Context(), `INSERT INTO app_namespaces(id,owner_id,handle) VALUES($1,$2,$3)`, id(), owner, *input.UserSlug)
			return slugConflict(err)
		}
		if err != nil || old == *input.UserSlug {
			return err
		}
		var occupied bool
		if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM app_alias_history h JOIN apps a ON a.id=h.app_id WHERE h.namespace=$1 AND a.owner_id<>$2)`, *input.UserSlug, owner).Scan(&occupied); err != nil {
			return err
		}
		if occupied {
			return fail(409, "slug_taken", "This user slug has historical links")
		}
		rows, err := tx.Query(r.Context(), `SELECT id,slug,host_label FROM apps WHERE namespace_id=$1 ORDER BY id`, namespaceID)
		if err != nil {
			return err
		}
		type item struct {
			id, slug  string
			hostLabel *string
		}
		var apps []item
		for rows.Next() {
			var a item
			if err = rows.Scan(&a.id, &a.slug, &a.hostLabel); err != nil {
				rows.Close()
				return err
			}
			apps = append(apps, a)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, a := range apps {
			validHost := validAppSlug(a.slug, *input.UserSlug)
			if !validHost && a.hostLabel != nil {
				return fail(400, "invalid_user_slug", "Shorten application slugs before choosing this user slug")
			}
			if validHost {
				if err = claimHost(r, tx, a.slug+"-"+*input.UserSlug, a.id); err != nil {
					return err
				}
			}
			_, err = tx.Exec(r.Context(), `INSERT INTO app_alias_history(namespace,slug,app_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, old, a.slug, a.id)
			if err != nil {
				return err
			}
			if validHost {
				_, err = tx.Exec(r.Context(), `UPDATE apps SET host_label=$1 WHERE id=$2`, a.slug+"-"+*input.UserSlug, a.id)
				if err != nil {
					return slugConflict(err)
				}
			}
		}
		_, err = tx.Exec(r.Context(), `UPDATE app_namespaces SET handle=$1 WHERE id=$2`, *input.UserSlug, namespaceID)
		return slugConflict(err)
	})
	if err != nil {
		return err
	}
	return s.webProfile(w, r, owner)
}
