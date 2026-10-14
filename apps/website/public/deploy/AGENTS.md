# Deploy to Cellapp — instructions for coding agents

The user wants the current project developed and deployed to Cellapp. Use the official `cellapp` CLI. Handle project inspection, CLI setup, building, upload and verification yourself. The user only needs to approve device authorization in the browser. Do not implement Cellapp's upload protocol yourself.

## Inspect the project

Work from the user's project directory. Read its package manifest, lockfile, build scripts, static output and any existing `cellapp.json`. Determine a working build command and the output directory containing `index.html`. For plain HTML, omit the build command. Add `--spa` only for client-side routes. Reuse the association in `cellapp.json` on later deployments. Ask only when the build command or output directory is genuinely ambiguous.

Cellapp hosts static frontend files. It does not run Next.js SSR, a custom server, serverless functions or a private backend. If the project needs a server, explain the blocker or prepare a genuine static export when that preserves the intended behavior. Do not put private credentials in bundled frontend files; admitted visitors can read them.

## Prepare the CLI

Check whether `cellapp --help` works. If it does, use that CLI. Otherwise check Node.js is at least 22.12 and npm is available. Then install and verify the published package:

```sh
npm install -g @cellapp/cli@latest
cellapp --help
```

If global installation is blocked, use `npm exec --yes --package=@cellapp/cli@latest -- cellapp --help` and the same `npm exec --yes --package=@cellapp/cli@latest -- cellapp` prefix for later commands. Do not default to asking the user to install the CLI. If Node.js or npm is unavailable, the package cannot be fetched, or both CLI invocation methods fail, report the specific blocker. Do not use `sudo`, disable TLS verification or install from an unverified source.

The CLI's default control origin is `https://console.cellapp.ai`. Respect an origin supplied by the user, the existing project, or `CELLAPP_ORIGIN`; do not substitute the public website `https://cellapp.ai` as an API origin. Keep credentials out of project files and logs.

## Build, authorize and publish

Use the project's actual output directory and build command. For example:

```sh
cellapp deploy --project . --output dist --build 'npm run build'
```

Omit `--build` for plain HTML. The CLI builds locally, validates static files, uploads them and publishes atomically. If device authorization starts, send the verification URL and code to the user and keep the command running while the user approves in the browser. Never request a password, cookie, access key or deployment credential. A chat reply is not browser approval.

Report success only when the CLI returns `status: published`. Give the user the app URL and explain any remaining visitor access step. The first deployment may return an access key once: share it only with the requesting owner, never put it in a URL, repository, issue or project file. A public app still requires visitors to sign in to Cellapp; a restricted app also needs owner approval or key redemption.

## Recover and extend

For a failed build, fix the project and rerun the same command. For a transient upload or network failure, retry that command once; pending deployment state is reusable. After another failure, report the error and current state. Do not delete the app, change slugs or invent a new association to hide failure. For `app_data_not_ready`, inspect `cellapp data status --project .` and use `cellapp data reconcile --project .` after the service is healthy before retrying.

For persistent application tables, follow the [Cellapp deployment Skill](https://github.com/cellapp-ai/skills/blob/main/skills/cellapp-deploy/SKILL.md) for the separate plan, apply and verify flow. Browser-only local state does not synchronize across devices. Cellapp currently provides static publishing and controlled application tables. Checkout and paid app sales are not part of this deployment flow.
