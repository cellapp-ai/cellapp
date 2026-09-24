# CellApp server

This repository contains the CellApp Go control API, protected static hosting gateway, database migrations, and local development infrastructure. The CLI and `cellapp-deploy` Agent Skill live in [cellapp-ai/skills](https://github.com/cellapp-ai/skills). This repository does not contain production credentials or internal planning files.

## Local development

Requires Go 1.24+, Node.js 22.12+ for the optional browser test, and Docker Compose.

```sh
go mod download
docker compose -f infra/compose.yaml up -d
cp .env.example .env
```

For local storage tests, configure a private S3-compatible bucket and inject `QINIU_ACCESS_KEY` and `QINIU_SECRET_KEY` through the process environment. Do not commit credentials or a populated `.env` file. Load the environment file you maintain, then migrate and start the service:

```sh
set -a
. ./.env
set +a
go run ./apps/server/cmd/server -migrate
go run ./apps/server/cmd/server
```

For production, use `AUTH_MODE=github` with a GitHub OAuth App whose callback is `${CONTROL_ORIGIN}/auth/callback`. Local `AUTH_MODE=dev` is limited to a nonproduction loopback control origin. Keep production credentials, database backups and object storage access outside this repository. Back up the database and private bucket together before migrations; do not use `docker compose down -v` as a rollback procedure.

Existing local database names and the internal development identity issuer retain their historical values to preserve deployed data. The public product and CLI name is CellApp.

## Validation

```sh
go test ./apps/server/...
go vet ./apps/server/...
go build -o dist/cellapp-server ./apps/server/cmd/server
```

The server pins the public client contract in [contracts/http-v1.json](contracts/http-v1.json) and records the supported version in [contracts/client-support.json](contracts/client-support.json). Database integration tests require `TEST_DATABASE_URL`. The optional browser test requires `RUN_BROWSER_TESTS=1`, `TEST_DATABASE_URL`, a Chrome executable, Node dependencies from `npm ci`, and `CELLAPP_CLI_ENTRY` pointing to the compiled entry of the approved public CLI package. It does not import public repository source.

The service is limited to protected static HTML, JavaScript, CSS and assets. It does not host application backends, synchronize browser data or offer public anonymous access.
