#!/usr/bin/env python3
"""Single-host Cellapp bootstrap. Secrets stay in infra/selfhost.local."""

import argparse
import json
import os
import re
import secrets
import shutil
import socket
import ssl
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
CONFIG = ROOT / "infra" / "selfhost.local"
COMPOSE = ROOT / "infra" / "compose.selfhost.yaml"
DOMAIN = re.compile(r"^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)+$")
OAUTH = re.compile(r"^[A-Za-z0-9_-]+$")
SECRET_NAMES = ("POSTGRES_PASSWORD", "CONTROL_DB_PASSWORD", "APP_DATA_DB_PASSWORD", "POSTGREST_DB_PASSWORD", "SECRET", "APP_DATA_JWT_SECRET", "MINIO_ROOT_USER", "MINIO_ROOT_PASSWORD")


def fail(message):
    raise ValueError(message)


def run(command, *, input_text=None):
    result = subprocess.run(command, input=input_text, text=True, capture_output=True, check=False)
    if result.returncode:
        fail(f"command failed: {command[0]} {command[1] if len(command) > 1 else ''}")
    return result.stdout


def write_private(path, content):
    temporary = path.with_name(path.name + ".tmp")
    with temporary.open("w", encoding="utf-8") as output:
        output.write(content)
    temporary.chmod(0o600)
    temporary.replace(path)


def cert_matches(cert, key, host):
    try:
        certificate = ssl._ssl._test_decode_cert(str(cert))
        ssl.match_hostname(certificate, host)
        expires = datetime.strptime(certificate["notAfter"], "%b %d %H:%M:%S %Y %Z").replace(tzinfo=timezone.utc)
        if expires <= datetime.now(timezone.utc):
            fail(f"TLS certificate expired for {host}")
    except (ssl.CertificateError, ValueError):
        fail(f"TLS certificate does not cover {host}")
    cert_key = run(["openssl", "x509", "-in", str(cert), "-pubkey", "-noout"])
    private_key = run(["openssl", "pkey", "-in", str(key), "-pubout"])
    if cert_key != private_key:
        fail(f"TLS certificate and key do not match for {host}")


def setup(args):
    old = json.loads((CONFIG / "settings.json").read_text()) if (CONFIG / "settings.json").exists() else {}
    control = args.control_host or old.get("control_host")
    apps = args.apps_domain or old.get("apps_domain")
    if not control or not apps or not DOMAIN.fullmatch(control) or not DOMAIN.fullmatch(apps):
        fail("supply valid --control-host and --apps-domain DNS names")
    if control == apps or control.endswith("." + apps) or apps.endswith("." + control):
        fail("control host and application domain overlap")
    # Conservative preflight; the server additionally checks the public suffix list.
    if control.split(".")[-2:] == apps.split(".")[-2:]:
        fail("control host and application domain must use different registrable domains")
    client_id = args.github_client_id or old.get("github_client_id")
    if not client_id or not OAUTH.fullmatch(client_id) or client_id.lower().startswith(("your_", "replace", "example", "development")):
        fail("supply a valid GitHub OAuth Client ID")
    client_secret = (Path(args.github_client_secret_file).read_text().strip() if args.github_client_secret_file else old.get("github_client_secret"))
    if not client_secret or not OAUTH.fullmatch(client_secret) or client_secret.lower().startswith(("your_", "replace", "example", "development")):
        fail("supply a valid GitHub OAuth Client secret file")
    paths = {}
    for name in ("control_cert", "control_key", "apps_cert", "apps_key"):
        supplied = getattr(args, name)
        path = Path(supplied).expanduser().resolve() if supplied else CONFIG / (name.replace("_cert", ".crt").replace("_key", ".key"))
        if not path.is_file():
            fail(f"missing TLS file: {name}")
        paths[name] = path
    cert_matches(paths["control_cert"], paths["control_key"], control)
    cert_matches(paths["apps_cert"], paths["apps_key"], "probe." + apps)

    CONFIG.mkdir(mode=0o700, parents=True, exist_ok=True)
    CONFIG.chmod(0o700)
    values = json.loads((CONFIG / "secrets.json").read_text()) if (CONFIG / "secrets.json").exists() else {}
    for name in SECRET_NAMES:
        if name not in values:
            values[name] = secrets.token_urlsafe(48)
        if len(values[name]) < 32 or "development" in values[name].lower() or len(set(values[name])) < 8:
            fail(f"unsafe generated secret: {name}")
    if len(set(values.values())) != len(values):
        fail("production secrets must be distinct")
    if not (CONFIG / "secrets.json").exists():
        write_private(CONFIG / "secrets.json", json.dumps(values, indent=2) + "\n")
    for name, path in paths.items():
        destination = CONFIG / (name.replace("_cert", ".crt").replace("_key", ".key"))
        if path != destination:
            shutil.copyfile(path, destination)
            destination.chmod(0o600)
    if not (CONFIG / "minio-ca.crt").exists():
        cert_dir = CONFIG / "minio-certs"
        cert_dir.mkdir(mode=0o700, exist_ok=True)
        ca_key = CONFIG / "minio-ca.key"
        run(["openssl", "req", "-x509", "-newkey", "rsa:3072", "-nodes", "-days", "3650", "-subj", "/CN=Cellapp private MinIO CA", "-keyout", str(ca_key), "-out", str(CONFIG / "minio-ca.crt")])
        csr = CONFIG / "minio.csr"
        run(["openssl", "req", "-newkey", "rsa:3072", "-nodes", "-subj", "/CN=minio", "-keyout", str(cert_dir / "private.key"), "-out", str(csr)])
        ext = CONFIG / "minio.ext"
        write_private(ext, "subjectAltName=DNS:minio\nextendedKeyUsage=serverAuth\n")
        run(["openssl", "x509", "-req", "-in", str(csr), "-CA", str(CONFIG / "minio-ca.crt"), "-CAkey", str(ca_key), "-CAcreateserial", "-days", "825", "-extfile", str(ext), "-out", str(cert_dir / "public.crt")])
        csr.unlink()
        ext.unlink()
        for file in (ca_key, CONFIG / "minio-ca.crt", cert_dir / "private.key", cert_dir / "public.crt"):
            file.chmod(0o600)
    # The application runs as a non-root user and only reads the public CA.
    (CONFIG / "minio-ca.crt").chmod(0o644)
    settings = dict(control_host=control, apps_domain=apps, github_client_id=client_id, github_client_secret=client_secret)
    write_private(CONFIG / "settings.json", json.dumps(settings, indent=2) + "\n")
    environment = {
        "NODE_ENV": "production", "AUTH_MODE": "github", "CONTROL_ORIGIN": "https://" + control,
        "APPS_DOMAIN": apps, "GITHUB_CLIENT_ID": client_id, "GITHUB_CLIENT_SECRET": client_secret,
        "DATABASE_URL": f"postgres://cellapp_control:{values['CONTROL_DB_PASSWORD']}@postgres:5432/cellapp?sslmode=disable",
        "APP_DATA_DATABASE_URL": f"postgres://cellapp_appdata_provisioner:{values['APP_DATA_DB_PASSWORD']}@postgres:5432/cellapp_data?sslmode=disable",
        "POSTGREST_URL": "http://127.0.0.1:3001", "PGRST_SERVER_HOST": "127.0.0.1", "PGRST_SERVER_PORT": "3001",
        "PGRST_DB_URI": f"postgres://cellapp_appdata_authenticator:{values['POSTGREST_DB_PASSWORD']}@postgres:5432/cellapp_data?sslmode=disable",
        "PGRST_DB_SCHEMAS": "cellapp_api", "PGRST_DB_PRE_CONFIG": "cellapp_internal.pre_config",
        "PGRST_DB_ANON_ROLE": "cellapp_appdata_anon", "PGRST_OPENAPI_MODE": "disabled", "PGRST_DB_POOL": "5",
        "PGRST_DB_MAX_ROWS": "1000", "PGRST_DB_PRE_REQUEST": "cellapp_internal.pre_request",
        "PGRST_JWT_SECRET": values["APP_DATA_JWT_SECRET"],
        "S3_ENDPOINT": "https://minio:9000", "S3_REGION": "us-east-1", "S3_BUCKET": "cellapp",
        "S3_ACCESS_KEY_ID": values["MINIO_ROOT_USER"], "S3_SECRET_ACCESS_KEY": values["MINIO_ROOT_PASSWORD"],
        "MAX_APPS": "10", "MAX_FILES": "1000", "MAX_FILE_BYTES": "10000000", "MAX_DEPLOYMENT_BYTES": "50000000",
        "MAX_STORAGE_BYTES": "500000000", "MAX_MONTHLY_VISITORS": "1000", "MAX_REQUESTS_PER_MINUTE": "120",
        "UPLOAD_TTL_SECONDS": "3600", "RETENTION_SECONDS": "86400",
        **values,
    }
    write_private(CONFIG / "runtime.env", "".join(f"{key}={value}\n" for key, value in environment.items()))
    write_private(CONFIG / "Caddyfile", f"""{{\n  auto_https off\n  admin off\n}}\nhttp://{control} {{\n  redir https://{control}{{uri}} permanent\n}}\nhttp://*.{apps} {{\n  redir https://{{host}}{{uri}} permanent\n}}\nhttps://{control} {{\n  tls /etc/caddy/control.crt /etc/caddy/control.key\n  reverse_proxy app:8080 {{\n    header_up Host {{http.request.host}}\n  }}\n}}\nhttps://*.{apps} {{\n  tls /etc/caddy/apps.crt /etc/caddy/apps.key\n  reverse_proxy app:8080 {{\n    header_up Host {{http.request.host}}\n  }}\n}}\n""")
    print("Self-host configuration prepared in infra/selfhost.local (secrets omitted).")


def compose(*arguments):
    subprocess.run(["docker", "compose", "-f", str(COMPOSE), *arguments], cwd=ROOT, check=True)


def check_ingress():
    control = json.loads((CONFIG / "settings.json").read_text())["control_host"]
    trust = ssl.create_default_context()
    trust.load_verify_locations(cafile=str(CONFIG / "control.crt"))
    with socket.create_connection(("127.0.0.1", 443), timeout=5) as connection:
        with trust.wrap_socket(connection, server_hostname=control) as tls:
            tls.sendall(f"GET /health HTTP/1.1\r\nHost: {control}\r\nConnection: close\r\n\r\n".encode())
            if b" 200 " not in tls.makefile("rb").readline(1024):
                fail("public TLS ingress health request did not return 200")


def start(_args):
    if not (CONFIG / "runtime.env").is_file():
        fail("run setup before start")
    stages = [
        ("database and MinIO", ("up", "-d", "--wait", "postgres", "minio")),
        ("application databases", ("run", "--rm", "database-init")),
        ("private bucket", ("run", "--rm", "minio-init")),
        ("control migrations", ("run", "--rm", "migrate")),
        ("application health", ("up", "-d", "--no-deps", "--force-recreate", "--wait", "app")),
        ("public TLS ingress", ("up", "-d", "--no-deps", "--force-recreate", "--wait", "ingress")),
    ]
    for label, command in stages:
        print(f"Checking {label}...", flush=True)
        try:
            compose(*command)
        except subprocess.CalledProcessError:
            fail(f"{label} failed; volumes retained. Fix the error and rerun: python3 scripts/selfhost.py start")
    try:
        check_ingress()
    except (OSError, ssl.SSLError):
        fail("public TLS ingress failed its local certificate and health check; volumes retained")
    print("Cellapp stack is running. Verify public DNS, HTTPS and GitHub login from a browser.")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    subs = parser.add_subparsers(dest="command", required=True)
    setup_command = subs.add_parser("setup")
    for flag in ("control-host", "apps-domain", "github-client-id", "github-client-secret-file", "control-cert", "control-key", "apps-cert", "apps-key"):
        setup_command.add_argument("--" + flag)
    subs.add_parser("start")
    args = parser.parse_args()
    try:
        {"setup": setup, "start": start}[args.command](args)
    except (OSError, ValueError) as error:
        print(f"Self-host {args.command} failed: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
