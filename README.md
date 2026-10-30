# Cellapp — Your Personal AI App Workspace

[English](README.md) | [简体中文](README.zh-CN.md)

Build apps with an Agent, then publish, manage, and share them.

For the in-progress personal model and local setup features, see the [Chinese usage guide](docs/personal-agent-guide.md). They are not released to the live service or npm yet.

An Agent can build your app on your computer and publish it to Cellapp with the deployment Skill. You can also publish with the `cellapp` CLI. Each app gets a stable HTTPS address that you can keep updating, control access to, and connect to its own data tables when it needs shared data. Cellapp delivers the app and handles sign-in and data access; it does not run uploaded programs.

## Quick start

### With an Agent (recommended)

Send this one request to the coding Agent working on your project:

> Install the Cellapp deployment Skill with `npx skills add cellapp-ai/skills`, then follow it to build and deploy this project to Cellapp. Use app data tables if needed, and give me the published URL.

The Agent installs the Skill, obtains the official CLI when needed, and handles the build and deployment. You only need to approve Cellapp access in your browser on first use. If your Agent does not support Skills, give it the [public Agent instructions](https://cellapp.ai/deploy/AGENTS.md) instead.

### Manually with the CLI

If you prefer to work in a terminal, install the CLI globally and verify it. Node.js 22.12 or later and npm are required.

```sh
npm install -g @cellapp/cli@latest
cellapp --help
```

Run `cellapp login` to connect to Cellapp, then `cellapp deploy --help` for deployment options. Build output and authorization files stay on your computer; deployment credentials are not included in the published app. See the [client documentation](skills/README.md) for the full CLI and app data workflow.

## Build apps without setting up a server

Cellapp provides account sign-in, access control, and separate data tables for each app. An Agent can build and publish locally; the app runs in the browser and reads and writes shared data through controlled APIs. You can focus on the interface and interactions without setting up your own server for publishing, user access, or data storage.

Apps that need continuously running background jobs or custom server-side logic still need a separately deployed service.

## Self-host Cellapp

The [single-host Docker Compose guide](docs/self-hosting.md) (Chinese) covers the full platform: the Go service, dashboard, separate app data API, PostgreSQL, private RustFS by default, and an HTTPS entry point. You will need a control domain and an app wildcard domain under different registrable domains, trusted certificates for both, and a GitHub OAuth app. For the existing Cloud Run deployment path, see the [operations guide](docs/operations.md) (Chinese).

## Contribute

The [development guide](docs/development.md) (Chinese) covers checkout, local dependencies, builds, tests, and contribution conventions. The public CLI and Skill live in the separate `skills` Git submodule; the Web dashboard and Go service live in this repository.

## License

Original code and documentation in this repository are licensed under the [Apache License 2.0](LICENSE); see [NOTICE](NOTICE) for copyright notices. The separate `skills` submodule has its own [MIT License](skills/LICENSE).
