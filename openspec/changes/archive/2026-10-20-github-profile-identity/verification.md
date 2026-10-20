# Verification

- Passed: `npm run typecheck --workspace @cellapp/web`, `npm run lint:web`, and `npm test --workspace @cellapp/web`. The built-artifact browser suite checked the GitHub name/avatar, live name edit, avatar fallback, 320–1440 px layouts, keyboard behavior and axe.
- Passed: focused GitHub OAuth, profile and slug tests under `go test -race` against disposable PostgreSQL; full `go test -race -p 1 -count=1 ./apps/server/...` and `go vet ./apps/server/...`.
- Passed: `RUN_BROWSER_TESTS=1 ... go test -race -count=1 -run '^TestBrowserEndToEnd$' -v ./apps/server/internal/hosting` with isolated HTTPS Chrome, PostgreSQL, PostgREST and MinIO. Its first attempt received `app_data_not_ready` from the disposable PostgREST cache; restarting that test container and rerunning passed.
- Passed: `openspec validate github-profile-identity --strict` and `git diff --check`.
- Not run: production GitHub sign-in or Cloud Run deployment in this change. Existing GitHub sessions without stored login need the next sign-in to populate the new default; existing user slugs are deliberately retained.
