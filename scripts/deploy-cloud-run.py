#!/usr/bin/env python3
"""Build and release the Cellapp service to an existing Cloud Run environment."""

import argparse
import json
import re
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.parse import urlparse
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parents[1]
NAME = re.compile(r"^[a-z][a-z0-9-]{1,61}[a-z0-9]$")
DOMAIN = re.compile(r"^(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}$")
DIGEST = re.compile(r"^sha256:[0-9a-f]{64}$")
SECRET_KEYS = ("SECRET", "DATABASE_URL", "APP_DATA_DATABASE_URL", "APP_DATA_JWT_SECRET", "PGRST_DB_URI", "PGRST_JWT_SECRET", "GITHUB_CLIENT_SECRET", "S3_ACCESS_KEY_ID", "S3_SECRET_ACCESS_KEY")
OPTIONAL_SECRET_KEYS = ("MODEL_ENCRYPTION_KEY",)
PUBLIC_ENV = ("NODE_ENV", "AUTH_MODE", "CONTROL_ORIGIN", "APPS_DOMAIN", "GITHUB_CLIENT_ID", "S3_ENDPOINT", "S3_REGION", "S3_BUCKET", "POSTGREST_URL", "PGRST_SERVER_HOST", "PGRST_SERVER_PORT", "PGRST_DB_SCHEMAS", "PGRST_DB_PRE_CONFIG", "PGRST_DB_ANON_ROLE", "PGRST_OPENAPI_MODE", "PGRST_DB_POOL", "PGRST_DB_MAX_ROWS", "PGRST_DB_PRE_REQUEST", "MAX_APPS", "MAX_FILES", "MAX_FILE_BYTES", "MAX_DEPLOYMENT_BYTES", "MAX_STORAGE_BYTES", "MAX_MONTHLY_VISITORS", "MAX_REQUESTS_PER_MINUTE", "UPLOAD_TTL_SECONDS", "RETENTION_SECONDS", "PUBLICATION_PAUSED", "ALLOW_ANONYMOUS_APP_DATA_WRITES")
LIMIT_KEYS = ("MAX_APPS", "MAX_FILES", "MAX_FILE_BYTES", "MAX_DEPLOYMENT_BYTES", "MAX_STORAGE_BYTES", "MAX_MONTHLY_VISITORS", "MAX_REQUESTS_PER_MINUTE", "UPLOAD_TTL_SECONDS", "RETENTION_SECONDS")


class DeploymentError(Exception):
    pass


def required(data, key):
    value = data.get(key)
    if not isinstance(value, str) or not value.strip() or value != value.strip():
        raise DeploymentError(f"invalid or missing configuration: {key}")
    return value


def load_config(path):
    try:
        config = json.loads(path.read_text())
    except (OSError, ValueError) as exc:
        raise DeploymentError(f"cannot read deployment configuration: {exc}") from exc
    if not isinstance(config, dict):
        raise DeploymentError("deployment configuration must be a JSON object")
    for key in ("project", "region", "service", "repository", "service_account", "control_origin", "apps_domain", "github_client_id", "s3_endpoint", "s3_region", "s3_bucket", "scheduler_job", "scheduler_region"):
        required(config, key)
    if "vpc_connector" in config:
        required(config, "vpc_connector")
        if "network" in config or "subnet" in config:
            raise DeploymentError("choose either a VPC connector or a Direct VPC network and subnet")
    else:
        required(config, "network")
        required(config, "subnet")
    for key in ("project", "service", "repository", "scheduler_job"):
        if not NAME.fullmatch(config[key]):
            raise DeploymentError(f"invalid configuration: {key}")
    try:
        origin = urlparse(config["control_origin"])
        origin_port = origin.port
    except ValueError as exc:
        raise DeploymentError("invalid control_origin") from exc
    if origin.scheme != "https" or not origin.hostname or origin.path or origin.query or origin.fragment or origin.username or origin.password or origin_port:
        raise DeploymentError("control_origin must be an HTTPS origin without a port or path")
    if not DOMAIN.fullmatch(origin.hostname) or not DOMAIN.fullmatch(config["apps_domain"]):
        raise DeploymentError("invalid production domain")
    if origin.hostname == config["apps_domain"] or origin.hostname.endswith("." + config["apps_domain"]):
        raise DeploymentError("control and app domains overlap")
    s3 = urlparse(config["s3_endpoint"])
    if s3.scheme != "https" or not s3.hostname or s3.path or s3.query or s3.fragment or s3.username or s3.password:
        raise DeploymentError("s3_endpoint must be an HTTPS endpoint")
    limits = config.get("limits")
    if not isinstance(limits, dict) or set(limits) != set(LIMIT_KEYS):
        raise DeploymentError("limits must contain all production quota and retention values")
    for key in LIMIT_KEYS:
        value = limits[key]
        if not isinstance(value, int) or isinstance(value, bool) or value <= 0 or value > 1 << 50:
            raise DeploymentError(f"invalid production limit: {key}")
    if not limits["MAX_FILE_BYTES"] <= limits["MAX_DEPLOYMENT_BYTES"] <= limits["MAX_STORAGE_BYTES"]:
        raise DeploymentError("file, deployment, and storage limits must be ascending")
    secrets = config.get("secrets")
    if not isinstance(secrets, dict) or not set(SECRET_KEYS).issubset(secrets) or set(secrets) - set(SECRET_KEYS) - set(OPTIONAL_SECRET_KEYS):
        raise DeploymentError("secrets must contain required Secret Manager references and only supported optional references")
    for key in secrets:
        ref = secrets[key]
        if not isinstance(ref, str) or not re.fullmatch(r"[A-Za-z][A-Za-z0-9_-]{0,254}:(?:latest|[1-9][0-9]*)", ref):
            raise DeploymentError(f"invalid secret reference: {key}")
    edge = config.get("cloudflare_edge")
    if edge is not None:
        if not isinstance(edge, dict) or set(edge) != {"origin_host", "secret"}:
            raise DeploymentError("cloudflare_edge requires origin_host and secret")
        if not isinstance(edge["origin_host"], str) or not DOMAIN.fullmatch(edge["origin_host"]):
            raise DeploymentError("invalid cloudflare_edge origin_host")
        if edge["origin_host"] == origin.hostname or edge["origin_host"].endswith("." + config["apps_domain"]):
            raise DeploymentError("cloudflare_edge origin_host overlaps a public domain")
        if not isinstance(edge["secret"], str) or not re.fullmatch(r"[A-Za-z][A-Za-z0-9_-]{0,254}:(?:latest|[1-9][0-9]*)", edge["secret"]):
            raise DeploymentError("invalid cloudflare_edge secret reference")
    delivery = config.get("edge_delivery")
    if delivery is not None:
        if not edge or not isinstance(delivery, dict) or set(delivery) != {"grant_key_id", "signing_key"}:
            raise DeploymentError("edge_delivery requires cloudflare_edge, grant_key_id and signing_key")
        if not isinstance(delivery["grant_key_id"], str) or not re.fullmatch(r"[A-Za-z0-9_-]{1,32}", delivery["grant_key_id"]):
            raise DeploymentError("invalid edge_delivery grant_key_id")
        if not isinstance(delivery["signing_key"], str) or not re.fullmatch(r"[A-Za-z][A-Za-z0-9_-]{0,254}:(?:latest|[1-9][0-9]*)", delivery["signing_key"]):
            raise DeploymentError("invalid edge_delivery signing_key reference")
        if config["s3_region"] != "auto":
            raise DeploymentError("edge_delivery requires R2 S3 region auto")
        if limits["RETENTION_SECONDS"] <= 60:
            raise DeploymentError("edge_delivery requires retention longer than grant lifetime")
    if not isinstance(config.get("publication_paused", False), bool):
        raise DeploymentError("publication_paused must be a boolean")
    if not isinstance(config.get("allow_anonymous_app_data_writes", False), bool):
        raise DeploymentError("allow_anonymous_app_data_writes must be a boolean")
    if secrets["APP_DATA_JWT_SECRET"] != secrets["PGRST_JWT_SECRET"]:
        raise DeploymentError("APP_DATA_JWT_SECRET and PGRST_JWT_SECRET must reference the same secret version")
    if "cloudsql_instance" in config and not re.fullmatch(r"[a-z][a-z0-9-]+:[a-z0-9-]+:[a-z][a-z0-9-]+", config["cloudsql_instance"]):
        raise DeploymentError("invalid cloudsql_instance")
    if not re.fullmatch(r"[a-z][a-z0-9-]*@[a-z][a-z0-9-]*\.iam\.gserviceaccount\.com", config["service_account"]):
        raise DeploymentError("invalid service_account")
    for key in ("region", "scheduler_region", "vpc_connector", "network", "subnet"):
        if key not in config:
            continue
        if not re.fullmatch(r"[a-z0-9][a-z0-9-]*", config[key]):
            raise DeploymentError(f"invalid configuration: {key}")
    return config


def gcloud(stage, *args, json_output=False):
    command = ["gcloud", *args, "--quiet"]
    if json_output:
        command.append("--format=json")
    try:
        result = subprocess.run(command, cwd=ROOT, capture_output=True, text=True, check=False)
    except OSError as exc:
        raise DeploymentError(f"{stage} failed: gcloud unavailable ({exc})") from exc
    if result.returncode:
        if stage == "Cloud Run Jobs capability":
            raise DeploymentError("Google Cloud CLI lacks Cloud Run Jobs; update gcloud before deploying")
        raise DeploymentError(f"{stage} failed; inspect Google Cloud CLI logs and resource permissions")
    if json_output:
        try:
            return json.loads(result.stdout)
        except ValueError as exc:
            raise DeploymentError(f"{stage} returned invalid JSON") from exc
    return result.stdout.strip()


def preflight(c):
    project, region = c["project"], c["region"]
    gcloud("Cloud Run Jobs capability", "run", "jobs", "deploy", "--help")
    gcloud("project", "projects", "describe", project, f"--project={project}")
    gcloud("Artifact Registry", "artifacts", "repositories", "describe", c["repository"], f"--location={region}", f"--project={project}")
    gcloud("service account", "iam", "service-accounts", "describe", c["service_account"], f"--project={project}")
    if c.get("vpc_connector"):
        gcloud("VPC connector", "compute", "networks", "vpc-access", "connectors", "describe", c["vpc_connector"], f"--region={region}", f"--project={project}")
    else:
        gcloud("VPC network", "compute", "networks", "describe", c["network"], f"--project={project}")
        subnet = gcloud("VPC subnet", "compute", "networks", "subnets", "describe", c["subnet"], f"--region={region}", f"--project={project}", json_output=True)
        if subnet.get("network", "").rsplit("/", 1)[-1] != c["network"]:
            raise DeploymentError("Direct VPC subnet does not belong to the configured network")
    if c.get("cloudsql_instance"):
        gcloud("Cloud SQL instance", "sql", "instances", "describe", c["cloudsql_instance"].split(":")[-1], f"--project={project}")
    for key in c["secrets"]:
        name, version = c["secrets"][key].rsplit(":", 1)
        gcloud(f"secret reference {key}", "secrets", "versions", "describe", version, f"--secret={name}", f"--project={project}")
    if c.get("cloudflare_edge"):
        name, version = c["cloudflare_edge"]["secret"].rsplit(":", 1)
        gcloud("edge secret reference", "secrets", "versions", "describe", version, f"--secret={name}", f"--project={project}")
    if c.get("edge_delivery"):
        name, version = c["edge_delivery"]["signing_key"].rsplit(":", 1)
        gcloud("edge grant signing key reference", "secrets", "versions", "describe", version, f"--secret={name}", f"--project={project}")
    scheduler = gcloud("cleanup scheduler", "scheduler", "jobs", "describe", c["scheduler_job"], f"--location={c['scheduler_region']}", f"--project={project}", json_output=True)
    expected = f"/projects/{project}/locations/{region}/jobs/{c['service']}-cleanup:run"
    uri = scheduler.get("httpTarget", {}).get("uri", "")
    if expected not in uri or scheduler.get("state") != "ENABLED":
        raise DeploymentError("cleanup scheduler is not enabled or does not target the cleanup job")


def settings(c):
    env = {
        "NODE_ENV": "production", "AUTH_MODE": "github", "CONTROL_ORIGIN": c["control_origin"],
        "APPS_DOMAIN": c["apps_domain"], "GITHUB_CLIENT_ID": c["github_client_id"],
        "S3_ENDPOINT": c["s3_endpoint"], "S3_REGION": c["s3_region"], "S3_BUCKET": c["s3_bucket"],
        "POSTGREST_URL": "http://127.0.0.1:3001", "PGRST_SERVER_HOST": "127.0.0.1", "PGRST_SERVER_PORT": "3001",
        "PGRST_DB_SCHEMAS": "cellapp_api", "PGRST_DB_PRE_CONFIG": "cellapp_internal.pre_config",
        "PGRST_DB_ANON_ROLE": "cellapp_appdata_anon", "PGRST_OPENAPI_MODE": "disabled",
        "PGRST_DB_POOL": "5", "PGRST_DB_MAX_ROWS": "1000", "PGRST_DB_PRE_REQUEST": "cellapp_internal.pre_request",
    }
    env.update({key: str(c["limits"][key]) for key in LIMIT_KEYS})
    env["PUBLICATION_PAUSED"] = str(c.get("publication_paused", False)).lower()
    env["ALLOW_ANONYMOUS_APP_DATA_WRITES"] = str(c.get("allow_anonymous_app_data_writes", False)).lower()
    for key in PUBLIC_ENV:
        if "," in env[key] or "=" in env[key]:
            raise DeploymentError(f"unsupported character in configuration: {key}")
    secret_flags = ",".join(f"{key}={c['secrets'][key]}" for key in (*SECRET_KEYS, *OPTIONAL_SECRET_KEYS) if key in c["secrets"])
    network_flags = ([f"--vpc-connector={c['vpc_connector']}"] if c.get("vpc_connector")
                     else [f"--network={c['network']}", f"--subnet={c['subnet']}"])
    common = [f"--project={c['project']}", f"--region={c['region']}", f"--service-account={c['service_account']}",
              *network_flags, "--vpc-egress=private-ranges-only",
              "--set-env-vars=" + ",".join(f"{key}={env[key]}" for key in PUBLIC_ENV),
              "--set-secrets=" + secret_flags]
    return common


def service_settings(c):
    flags = settings(c)
    edge = c.get("cloudflare_edge")
    if edge:
        flags = [flag + ",EDGE_ORIGIN_HOST=" + edge["origin_host"] if flag.startswith("--set-env-vars=")
                 else flag + ",EDGE_ORIGIN_SECRET=" + edge["secret"] if flag.startswith("--set-secrets=")
                 else flag for flag in flags]
    delivery = c.get("edge_delivery")
    if delivery:
        flags = [flag + ",EDGE_GRANT_KEY_ID=" + delivery["grant_key_id"] if flag.startswith("--set-env-vars=")
                 else flag + ",EDGE_GRANT_SIGNING_KEY=" + delivery["signing_key"] if flag.startswith("--set-secrets=")
                 else flag for flag in flags]
    return flags


def image_digest(c):
    tag = f"{c['region']}-docker.pkg.dev/{c['project']}/{c['repository']}/cellapp:{c['source_revision']}"
    gcloud("Cloud Build", "builds", "submit", ".", f"--tag={tag}", "--ignore-file=.gcloudignore", f"--project={c['project']}", f"--region={c['region']}")
    image = gcloud("image lookup", "artifacts", "docker", "images", "describe", tag, f"--project={c['project']}", json_output=True)
    digest = image.get("image_summary", {}).get("digest") or image.get("imageSummary", {}).get("digest") or image.get("digest")
    if not isinstance(digest, str) or not DIGEST.fullmatch(digest):
        raise DeploymentError("image lookup did not return a valid digest")
    return tag.split(":")[0] + "@" + digest


def deploy_job(c, image, suffix, arg):
    name = c["service"] + suffix
    flags = settings(c)
    if c.get("cloudsql_instance"):
        flags.append("--set-cloudsql-instances=" + c["cloudsql_instance"])
    gcloud(f"{name} job deploy", "run", "jobs", "deploy", name, f"--image={image}",
           "--command=/usr/local/bin/cellapp-server", f"--args={arg}", "--tasks=1", "--max-retries=0", *flags)
    return name


def activate_edge_candidate(c, revision):
    gcloud("candidate tag", "run", "services", "update-traffic", c["service"],
           f"--update-tags=release-candidate={revision}", f"--project={c['project']}", f"--region={c['region']}")
    service = gcloud("candidate lookup", "run", "services", "describe", c["service"],
                     f"--project={c['project']}", f"--region={c['region']}", json_output=True)
    url = next((item.get("url") for item in service.get("status", {}).get("traffic", [])
                if item.get("tag") == "release-candidate" and item.get("revisionName") == revision), None)
    if not url:
        raise DeploymentError("candidate revision has no tagged test URL; previous traffic remains active")
    status, request_id = probe(url + "/health")
    if status != 200 or not request_id:
        raise DeploymentError("candidate health check failed; previous traffic remains active")
    active = gcloud("candidate readiness", "run", "revisions", "describe", revision,
                    f"--project={c['project']}", f"--region={c['region']}", json_output=True)
    conditions = {item.get("type"): item.get("status") for item in active.get("status", {}).get("conditions", [])}
    if any(conditions.get(key) != "True" for key in ("Ready", "Active", "ContainerHealthy", "ResourcesAvailable")):
        raise DeploymentError("candidate revision is not active and healthy; previous traffic remains active")


def release(c):
    preflight(c)
    services = gcloud("service inventory", "run", "services", "list",
                      f"--project={c['project']}", f"--region={c['region']}", json_output=True)
    if not isinstance(services, list):
        raise DeploymentError("service inventory returned invalid JSON")
    current = next((service for service in services if isinstance(service, dict)
                    and service.get("metadata", {}).get("name") == c["service"]), None)
    if current and not c.get("cloudflare_edge") and current.get("metadata", {}).get("annotations", {}).get("run.googleapis.com/ingress") == "all":
        raise DeploymentError("existing service uses public ingress; configure cloudflare_edge before releasing")
    flags = service_settings(c)
    source_state = subprocess.run(["git", "status", "--porcelain"], cwd=ROOT, capture_output=True, text=True, check=False)
    if source_state.returncode:
        raise DeploymentError("cannot inspect source worktree")
    if source_state.stdout.strip():
        print("Warning: deploying uncommitted source; retain the reported image digest for recovery", file=sys.stderr)
    source_result = subprocess.run(["git", "rev-parse", "--short=12", "HEAD"], cwd=ROOT, capture_output=True, text=True, check=False)
    if source_result.returncode:
        raise DeploymentError("cannot determine source revision")
    source = source_result.stdout.strip()
    c["source_revision"] = source + "-" + datetime.now(timezone.utc).strftime("%Y%m%d%H%M%S")
    image = image_digest(c)
    print(f"Built image: {image}")
    migration = deploy_job(c, image, "-migrate", "-migrate")
    gcloud("migration execution", "run", "jobs", "execute", migration, "--wait", f"--project={c['project']}", f"--region={c['region']}")
    migration_info = gcloud("migration execution lookup", "run", "jobs", "describe", migration, f"--project={c['project']}", f"--region={c['region']}", json_output=True)
    execution_name = migration_info.get("status", {}).get("latestCreatedExecution", {}).get("name")
    if not execution_name:
        raise DeploymentError("migration completed but execution ID could not be confirmed")
    execution = gcloud("migration result", "run", "jobs", "executions", "describe", execution_name, f"--project={c['project']}", f"--region={c['region']}", json_output=True)
    result = execution.get("status", execution)
    if result.get("succeededCount") != 1 or result.get("failedCount", 0) != 0 or not result.get("completionTime"):
        raise DeploymentError(f"migration execution {execution_name} did not report one successful task")
    print(f"Migration completed: {execution_name}")
    deploy_job(c, image, "-cleanup", "-cleanup")
    if c.get("cloudsql_instance"):
        flags.append("--set-cloudsql-instances=" + c["cloudsql_instance"])
    exists = current is not None
    previous_traffic = [f"{item['revisionName']}={item['percent']}" for item in
                        (current or {}).get("status", {}).get("traffic", [])
                        if item.get("revisionName") and item.get("percent", 0) > 0]
    deployed = gcloud("service deploy", "run", "deploy", c["service"], f"--image={image}",
                      "--ingress=" + ("all" if c.get("cloudflare_edge") else "internal-and-cloud-load-balancing"), "--port=8080", "--allow-unauthenticated",
                      "--no-cpu-throttling", "--min=1",
                      "--startup-probe=httpGet.path=/health,httpGet.port=8080,periodSeconds=5,timeoutSeconds=5,failureThreshold=24",
                      *(["--no-traffic"] if exists else []), *flags, json_output=True)
    revision = deployed.get("status", {}).get("latestCreatedRevisionName")
    if not revision:
        raise DeploymentError("service deploy did not report a revision")
    ready = gcloud("revision readiness", "run", "revisions", "describe", revision, f"--project={c['project']}", f"--region={c['region']}", json_output=True)
    revision_containers = ready.get("spec", {}).get("containers", [])
    if not revision_containers or revision_containers[0].get("image") != image:
        raise DeploymentError(f"revision {revision} does not use the expected image digest")
    conditions = ready.get("status", {}).get("conditions", [])
    if not any(item.get("type") == "Ready" and item.get("status") == "True" for item in conditions):
        raise DeploymentError(f"revision {revision} is not ready; previous traffic remains active")
    if c.get("cloudflare_edge"):
        activate_edge_candidate(c, revision)
    try:
        gcloud("traffic switch", "run", "services", "update-traffic", c["service"], f"--to-revisions={revision}=100", f"--project={c['project']}", f"--region={c['region']}")
    except DeploymentError as exc:
        if previous_traffic:
            try:
                gcloud("traffic rollback", "run", "services", "update-traffic", c["service"],
                       "--to-revisions=" + ",".join(previous_traffic),
                       f"--project={c['project']}", f"--region={c['region']}")
            except DeploymentError as rollback_error:
                raise DeploymentError("traffic switch and automatic rollback failed; inspect the service's desired traffic") from rollback_error
        raise DeploymentError("traffic switch failed; previous traffic restored" if previous_traffic else
                              "traffic switch failed; previous traffic could not be determined") from exc
    print(f"Revision serving traffic: {revision}")
    verify_routes(c)
    print(f"Control: {c['control_origin']}")
    print(f"Applications: https://*.{c['apps_domain']}")
    print("Cloud Run release and domain checks completed")


def probe(url):
    try:
        with urlopen(Request(url, headers={"User-Agent": "Cellapp-deploy-check"}), timeout=15) as response:
            return response.status, response.headers.get("X-Request-ID")
    except HTTPError as exc:
        return exc.code, exc.headers.get("X-Request-ID")
    except (OSError, URLError) as exc:
        raise DeploymentError(f"domain route unavailable: {urlparse(url).hostname} ({exc.__class__.__name__})") from exc


def verify_routes(c):
    control_status, control_id = probe(c["control_origin"] + "/dashboard/login")
    invalid_status, invalid_id = probe("https://cellapp-invalid-probe." + c["apps_domain"] + "/")
    # R2 edge delivery can return the origin's denial body without its request ID.
    if control_status != 200 or not control_id or invalid_status != 404 or (not c.get("edge_delivery") and not invalid_id):
        raise DeploymentError("custom domain checks failed; inspect edge routing and TLS; revision remains available for rollback")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", required=True, type=Path, help="path to non-secret deployment configuration JSON")
    args = parser.parse_args()
    try:
        config = load_config(args.config)
        release(config)
    except DeploymentError as exc:
        print(f"Deployment incomplete: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
