# Deploy to Cellapp — instructions for coding agents

[简体中文版](./AGENTS.zh-CN.md)

The user wants the current project developed and deployed to Cellapp. Run the official published client yourself for building, upload and verification. The user only needs to approve device authorization in the browser. Do not implement Cellapp's upload protocol yourself or ask the user to install a CLI.

## Inspect the project

Work from the user's project directory. Read its package manifest, lockfile, build scripts, static output and any existing `cellapp.json`. Determine a working build command and the output directory containing `index.html`. For plain HTML, omit the build command. Add `--spa` only for client-side routes. Reuse the association in `cellapp.json` on later deployments. Ask only when the build command or output directory is genuinely ambiguous.

Cellapp hosts static frontend files. It does not run Next.js SSR, a custom server, serverless functions or a private backend. If the project needs a server, explain the blocker or prepare a genuine static export when that preserves the intended behavior. Do not put private credentials in bundled frontend files; admitted visitors can read them.

## Run the published client on demand

Check Node.js is at least 22.12 and npm is available. Verify the official published package from the project directory:

```sh
npm exec --yes --package=@cellapp/cli@latest -- cellapp --help
```

Use the same `npm exec --yes --package=@cellapp/cli@latest -- cellapp` prefix for all later commands. npm obtains and caches the package on demand; no global install is needed. If Node.js or npm is unavailable, package retrieval fails, or help verification fails, report the specific blocker. Do not use `sudo`, disable TLS verification or install from an unverified source.

The client defaults to `https://cellapp.ai`. It migrates project associations at the exact retired origin `https://console.cellapp.ai` while preserving the application and pending deployment IDs. An explicit old origin is rejected; respect other custom origins supplied by the user or environment. The old credential is never copied: ask for browser approval once on this machine, then reuse the new user-level credential across the owner's projects. Keep credentials out of project files and logs.

## Authorize before long work

After verifying the client and selecting the project origin, check authorization before lengthy development or builds:

```sh
npm exec --yes --package=@cellapp/cli@latest -- cellapp apps --project .
```

This command reuses a saved credential or starts browser authorization when none exists. Send the verification URL and code to the user as soon as they appear, and keep the command running while they approve in the browser. If a saved credential is rejected with 401, run `cellapp login` with the same npm prefix and origin, then retry `cellapp apps`. If a code expires before approval, do not leave the old code as the next step: run login again when the user is ready and send the new URL and code. Never request a password, cookie, access key or deployment credential; a chat reply is not browser approval. Continue the long work after authorization succeeds.

## Develop the interface when needed

If the request includes developing or substantially redesigning an app and the user has not supplied UI direction, read the [user-app design guidance](https://cellapp.ai/deploy/app-design.md) and apply it as adaptable defaults without asking the user to opt in. Follow explicit user design instructions and the project's established conventions. Choose the framework and styling approach yourself from the app's needs; the guidance does not require a particular tool or Cellapp-branded look. If the request is only to deploy an existing app, continue the deployment flow without redesigning it. If the guidance is unavailable, use these precedence rules and continue the requested work.

## Build and publish

Use the project's actual output directory and build command. For example:

```sh
npm exec --yes --package=@cellapp/cli@latest -- cellapp deploy --project . --output dist --build 'npm run build'
```

Omit `--build` for plain HTML. The client builds locally, validates static files, uploads them and publishes atomically. If deployment still starts device authorization, send the new verification URL and code promptly and keep the command running while the user approves in the browser; if it expires, repeat authorization with a fresh code before retrying deployment.

Report success only when the CLI returns `status: published`. Give the user the app URL and explain any remaining visitor access step. The first deployment may return an access key once: share it only with the requesting owner, never put it in a URL, repository, issue or project file. A public app still requires visitors to sign in to Cellapp; a restricted app also needs owner approval or key redemption.

## Recover and extend

For a failed build, fix the project and rerun the same command. For a transient upload or network failure, retry that command once; pending deployment state is reusable. After another failure, report the error and current state. Do not delete the app, change slugs or invent a new association to hide failure. For `app_data_not_ready`, inspect `cellapp data status --project .` and use `cellapp data reconcile --project .` after the service is healthy before retrying.

For persistent application tables, follow the [Cellapp deployment Skill](https://github.com/cellapp-ai/skills/blob/main/skills/cellapp-deploy/SKILL.md) for the separate plan, apply and verify flow. Browser-only local state does not synchronize across devices. Cellapp currently provides static publishing and controlled application tables. Checkout and paid app sales are not part of this deployment flow.
