package hosting

import (
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"
)

type webProfileData struct {
	DisplayName    string `json:"displayName"`
	UserSlug       string `json:"userSlug"`
	GitHubUsername string `json:"githubUsername"`
}

func (s *Server) webProfile(w http.ResponseWriter, r *http.Request, owner string) error {
	var profile webProfileData
	if err := s.DB.QueryRow(r.Context(), `SELECT o.display_name,COALESCE(n.handle,''),o.github_login FROM owners o LEFT JOIN app_namespaces n ON n.owner_id=o.id WHERE o.id=$1`, owner).Scan(&profile.DisplayName, &profile.UserSlug, &profile.GitHubUsername); err != nil {
		return err
	}
	jsonResponse(w, http.StatusOK, profile)
	return nil
}

func (s *Server) webUpdateProfile(w http.ResponseWriter, r *http.Request, owner string) error {
	var input struct {
		DisplayName *string `json:"displayName"`
	}
	if err := decode(w, r, &input); err != nil {
		return err
	}
	if input.DisplayName == nil {
		return fail(400, "invalid_display_name", "Enter a display name or leave it empty")
	}
	name := strings.TrimSpace(*input.DisplayName)
	if name != *input.DisplayName || !utf8.ValidString(name) || utf8.RuneCountInString(name) > 80 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return fail(400, "invalid_display_name", "Use up to 80 characters without leading or trailing spaces or control characters")
	}
	if _, err := s.DB.Exec(r.Context(), `UPDATE owners SET display_name=$1 WHERE id=$2`, name, owner); err != nil {
		return err
	}
	return s.webProfile(w, r, owner)
}
