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

var reservedSlugs = map[string]bool{
	"accounts": true, "account": true, "admin": true, "api": true, "app": true,
	"apps": true, "assets": true, "auth": true, "billing": true, "cdn": true,
	"cloud": true, "console": true, "dashboard": true, "docs": true, "files": true,
	"help": true, "identity": true, "login": true, "logout": true, "metrics": true,
	"oauth": true, "static": true, "status": true, "support": true, "system": true,
	"www": true,
}

func validUserSlug(value string) bool {
	return len(value) >= 3 && len(value) <= 61 && aliasPart.MatchString(value) && !strings.HasSuffix(value, "-") && !reservedSlugs[value]
}

func validAppSlug(value, userSlug string) bool {
	return validUserSlug(userSlug) && aliasPart.MatchString(value) && !strings.HasSuffix(value, "-") && !reservedSlugs[value] && len(value)+1+len(userSlug) <= 63
}

func (s *Server) appURL(appID, label string) string {
	if label != "" {
		return s.Config.AppHostURL(label)
	}
	return s.Config.AppURL(appID)
}

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
		var issuer, subject, githubLogin string
		if err = tx.QueryRow(ctx, `SELECT issuer,subject,github_login FROM owners WHERE id=$1`, owner).Scan(&issuer, &subject, &githubLogin); err != nil {
			return "", "", err
		}
		candidates := []string{}
		if issuer == githubIssuer && githubLogin != "" {
			preferred := strings.ToLower(githubLogin)
			if !validUserSlug(preferred) {
				preferred = "u-" + preferred
			}
			if validUserSlug(preferred) {
				candidates = append(candidates, preferred)
			}
			candidates = append(candidates, "u-"+subject)
		}
		candidates = append(candidates, "n-"+namespaceID)
		for _, candidate := range candidates {
			if !validUserSlug(candidate) {
				continue
			}
			var used bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app_alias_history WHERE namespace=$1)`, candidate).Scan(&used); err != nil {
				return "", "", err
			}
			if used {
				continue
			}
			result, insertErr := tx.Exec(ctx, `INSERT INTO app_namespaces(id,owner_id,handle) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, namespaceID, owner, candidate)
			if insertErr != nil {
				return "", "", insertErr
			}
			if result.RowsAffected() == 1 {
				return namespaceID, candidate, nil
			}
		}
		return "", "", fail(409, "namespace_taken", "No available namespace for this account")
	}
	if !validUserSlug(handle) {
		return "", "", fail(400, "invalid_namespace", "User slug must be 3–61 lowercase characters, digits or hyphens and cannot be reserved")
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

func chooseAppSlug(ctx context.Context, tx pgx.Tx, namespaceID, userSlug, requested, name string) (string, error) {
	if !validUserSlug(userSlug) {
		return "", fail(400, "invalid_namespace", "Set a valid user slug before creating an application")
	}
	if requested != "" && !validAppSlug(requested, userSlug) {
		return "", fail(400, "invalid_slug", "Application slug is reserved, invalid or too long for the user slug")
	}
	base := requested
	if base == "" {
		base = slugBase(name)
		if reservedSlugs[base] {
			base += "-site"
		}
	}
	maxLength := 62 - len(userSlug)
	if len(base) > maxLength {
		base = strings.TrimRight(base[:maxLength], "-")
	}
	for number := 1; number <= 1000; number++ {
		slug := base
		if number > 1 {
			if requested != "" || maxLength <= len(fmt.Sprintf("-%d", number)) {
				return "", fail(409, "slug_taken", "Application alias is already in use")
			}
			suffix := fmt.Sprintf("-%d", number)
			slug = strings.TrimRight(base[:min(len(base), maxLength-len(suffix))], "-") + suffix
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM apps WHERE namespace_id=$1 AND slug=$2)
    OR EXISTS(SELECT 1 FROM app_alias_history h JOIN app_namespaces n ON n.handle=h.namespace WHERE n.id=$1 AND h.slug=$2)
    OR EXISTS(SELECT 1 FROM app_hostnames WHERE label=$3)`, namespaceID, slug, slug+"-"+userSlug).Scan(&exists); err != nil {
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
	var hostLabel string
	err := s.DB.QueryRow(r.Context(), `SELECT a.id,COALESCE(a.host_label,'') FROM apps a JOIN app_namespaces n ON n.id=a.namespace_id
    WHERE n.handle=$1 AND a.slug=$2 AND NOT a.deleted AND NOT a.suspended
    UNION ALL SELECT a.id,COALESCE(a.host_label,'') FROM app_alias_history h JOIN apps a ON a.id=h.app_id
    WHERE h.namespace=$1 AND h.slug=$2 AND NOT a.deleted AND NOT a.suspended LIMIT 1`, namespace, slug).Scan(&appID, &hostLabel)
	if err == pgx.ErrNoRows {
		return fail(404, "app_not_found", "Application not found")
	}
	if err != nil {
		return err
	}
	http.Redirect(w, r, s.appURL(appID, hostLabel), http.StatusTemporaryRedirect)
	return nil
}
