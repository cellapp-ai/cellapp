package hosting

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var aliasPart = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

func slugBase(name string) string {
	var out strings.Builder
	separator := false
	for _, char := range strings.ToLower(name) {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' {
			if separator && out.Len() > 0 && out.Len() < 40 {
				out.WriteByte('-')
			}
			separator = false
			if out.Len() < 40 {
				out.WriteRune(char)
			}
		} else {
			separator = true
		}
	}
	result := strings.Trim(out.String(), "-")
	if result == "" || result[0] < 'a' || result[0] > 'z' {
		return "app"
	}
	return result
}

func (s *Server) appAliasURL(namespace, slug string) string {
	return s.Config.ControlOrigin + "/a/" + namespace + "/" + slug
}

func ensureAppNamespace(ctx context.Context, tx pgx.Tx, owner, requested string) (string, string, error) {
	var namespaceID, handle string
	err := tx.QueryRow(ctx, `SELECT id,handle FROM app_namespaces WHERE owner_id=$1`, owner).Scan(&namespaceID, &handle)
	if err == nil {
		if requested != "" && requested != handle {
			return "", "", fail(409, "namespace_mismatch", "This account already has a different namespace")
		}
		return namespaceID, handle, nil
	}
	if err != pgx.ErrNoRows {
		return "", "", err
	}
	namespaceID = id()
	handle = requested
	if handle == "" {
		handle = "n-" + namespaceID
	}
	if !aliasPart.MatchString(handle) {
		return "", "", fail(400, "invalid_namespace", "Namespace must use lowercase letters, digits or hyphens")
	}
	_, err = tx.Exec(ctx, `INSERT INTO app_namespaces(id,owner_id,handle) VALUES($1,$2,$3)`, namespaceID, owner, handle)
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return "", "", fail(409, "namespace_taken", "Namespace is already in use")
		}
		return "", "", err
	}
	return namespaceID, handle, nil
}

func chooseAppSlug(ctx context.Context, tx pgx.Tx, namespaceID, requested, name string) (string, error) {
	if requested != "" && !aliasPart.MatchString(requested) {
		return "", fail(400, "invalid_slug", "Application alias must use lowercase letters, digits or hyphens")
	}
	base := requested
	if base == "" {
		base = slugBase(name)
	}
	for number := 1; number <= 1000; number++ {
		slug := base
		if number > 1 {
			if requested != "" {
				return "", fail(409, "slug_taken", "Application alias is already in use")
			}
			suffix := fmt.Sprintf("-%d", number)
			slug = base[:min(len(base), 63-len(suffix))] + suffix
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM apps WHERE namespace_id=$1 AND slug=$2)`, namespaceID, slug).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
	}
	return "", fail(409, "slug_taken", "No available application alias")
}

func (s *Server) redirectAppAlias(w http.ResponseWriter, r *http.Request) error {
	namespace, slug := r.PathValue("namespace"), r.PathValue("slug")
	if !aliasPart.MatchString(namespace) || !aliasPart.MatchString(slug) {
		return fail(404, "app_not_found", "Application not found")
	}
	var appID string
	err := s.DB.QueryRow(r.Context(), `SELECT a.id FROM apps a JOIN app_namespaces n ON n.id=a.namespace_id
    WHERE n.handle=$1 AND a.slug=$2 AND NOT a.deleted AND NOT a.suspended`, namespace, slug).Scan(&appID)
	if err == pgx.ErrNoRows {
		return fail(404, "app_not_found", "Application not found")
	}
	if err != nil {
		return err
	}
	http.Redirect(w, r, s.Config.AppURL(appID), http.StatusTemporaryRedirect)
	return nil
}
