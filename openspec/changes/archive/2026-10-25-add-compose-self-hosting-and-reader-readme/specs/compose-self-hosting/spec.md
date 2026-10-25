# Spec Delta

## Purpose

Provide a reproducible, secure single-host deployment of the complete Cellapp platform with Docker Compose and private RustFS storage by default, independent of the managed Cloud Run release path.

## ADDED Requirements

### Requirement: Complete Compose deployment

Cellapp SHALL provide a production Docker Compose path that starts the control service, matching Web console, application data API, control and application databases, private object storage, and HTTPS ingress from one documented source revision. Public application delivery SHALL work without Cloud Run, Cloudflare Workers, or an external S3 account.

#### Scenario: Fresh host deployment
- **WHEN** an operator supplies the documented host prerequisites and valid configuration, then runs the setup and start commands
- **THEN** all required services become healthy, the control login page loads, and an authorized static application can be published and opened through its application hostname

#### Scenario: Required service fails
- **WHEN** a database, object store, data API, migration, or ingress dependency fails
- **THEN** setup or health verification reports the failing stage and does not claim the platform is ready

### Requirement: Explicit production configuration

The Compose path SHALL require a control hostname, an application wildcard domain, trusted HTTPS for both, and GitHub OAuth credentials. It MUST generate or require distinct persistent production secrets for platform signing, application data signing, databases, and RustFS; it MUST reject development credentials and development authentication mode. The control hostname MUST NOT fall inside the application wildcard.

#### Scenario: One-domain deployment
- **WHEN** the control hostname is `example.com` and the application domain is `example.com` or `apps.example.com`
- **THEN** preflight accepts the layout, subject to valid DNS, certificates, OAuth credentials, and secrets

#### Scenario: Sibling or separate-domain deployment
- **WHEN** the control hostname is a sibling of the application domain or belongs to another registrable domain
- **THEN** preflight accepts the layout, subject to the same public configuration checks

#### Scenario: Invalid or missing configuration
- **WHEN** the control hostname falls inside the application wildcard, TLS material is absent, OAuth credentials are missing, or a secret retains a development default
- **THEN** the preflight or service startup fails before presenting the deployment as publicly usable

#### Scenario: Valid host routing
- **WHEN** a browser visits the configured control host or a published application subdomain over HTTPS
- **THEN** ingress preserves the requested host, routes it to the matching Cellapp surface, and rejects an unknown host without exposing another application's content

### Requirement: Private RustFS by default

The Compose path SHALL provision a durable private RustFS Bucket by default and connect the production service to it over verified HTTPS with non-development credentials. Object and management ports MUST NOT be publicly exposed. Operators MAY explicitly select a separately configured HTTPS S3-compatible store without changing application behavior.

#### Scenario: Repeat setup or restart
- **WHEN** the operator reruns initialization or restarts the stack after publishing files
- **THEN** the Bucket remains private and the published objects remain readable through authorized application delivery

#### Scenario: Unauthorized storage access
- **WHEN** an unauthenticated client attempts to read a stored object or reach the RustFS management endpoint from the public network
- **THEN** the object is not disclosed and the management endpoint is not available

### Requirement: Safe initialization and recovery

The self-hosting path SHALL run control and application database initialization and migrations before accepting publication, keep secrets and persistent data outside the image and repository, and document backup, update, failure recovery, and rollback. Repeated setup MUST preserve existing users, applications, database rows, and objects.

#### Scenario: Repeat or failed migration
- **WHEN** setup is repeated or a migration fails before the new service is ready
- **THEN** existing persistent volumes are retained, the failure is reported, and the operator has a documented retry path that does not recreate the installation

#### Scenario: Backup and rollback
- **WHEN** an operator backs up the stack or rolls back a bad application image
- **THEN** the guide identifies the required databases, object volume, and secret material, and explains migration compatibility limits without deleting persistent data

### Requirement: Self-hosting guide matches the working stack

Cellapp SHALL provide a dedicated self-hosting guide covering prerequisites, domain and TLS setup, OAuth, installation, verification, updates, backup, and recovery with commands that match the production Compose assets. The guide MUST distinguish completed checks from checks an operator still needs to run on real public domains.

#### Scenario: New operator follows the guide
- **WHEN** an operator starts from a clean checkout and follows the documented single-host path
- **THEN** each command names its expected result or a corrective action, ending with checks for login, publication, application access, and private storage
