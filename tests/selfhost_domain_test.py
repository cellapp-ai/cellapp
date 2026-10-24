"""Isolated setup checks for supported public host layouts."""

import argparse
import importlib.util
import json
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch


ROOT = Path(__file__).resolve().parents[1]
SPEC = importlib.util.spec_from_file_location("selfhost", ROOT / "scripts" / "selfhost.py")
selfhost = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(selfhost)


class SelfHostDomainTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.cert_dir = tempfile.TemporaryDirectory()
        cls.cert = Path(cls.cert_dir.name) / "public.crt"
        cls.key = Path(cls.cert_dir.name) / "public.key"
        subprocess.run(
            ["openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "2",
             "-subj", "/CN=example.com", "-addext",
             "subjectAltName=DNS:example.com,DNS:*.example.com,DNS:*.apps.example.com,DNS:control.example.net,DNS:*.apps.example.org",
             "-keyout", str(cls.key), "-out", str(cls.cert)],
            check=True, capture_output=True,
        )

    @classmethod
    def tearDownClass(cls):
        cls.cert_dir.cleanup()

    def args(self, control, apps):
        return argparse.Namespace(
            control_host=control, apps_domain=apps,
            github_client_id="validClient123", github_client_secret_file=None,
            control_cert=str(self.cert), control_key=str(self.key),
            apps_cert=str(self.cert), apps_key=str(self.key),
        )

    def test_accepted_layouts_and_repeat_setup(self):
        for control, apps in (
            ("example.com", "example.com"),
            ("example.com", "apps.example.com"),
            ("control.example.net", "apps.example.org"),
        ):
            with self.subTest(control=control, apps=apps), tempfile.TemporaryDirectory() as directory:
                config = Path(directory) / "local"
                args = self.args(control, apps)
                with patch.object(selfhost, "CONFIG", config):
                    # Supply a valid fake OAuth secret through a protected input file.
                    secret = Path(directory) / "oauth-secret"
                    secret.write_text("validOAuthSecret123456")
                    args.github_client_secret_file = str(secret)
                    selfhost.setup(args)
                    settings = json.loads((config / "settings.json").read_text())
                    self.assertEqual((settings["control_host"], settings["apps_domain"]), (control, apps))
                    caddy = (config / "Caddyfile").read_text()
                    self.assertIn("https://" + control + " {", caddy)
                    self.assertIn("https://*." + apps + " {", caddy)
                    originals = {name: (config / name).read_bytes() for name in
                                 ("secrets.json", "control.crt", "apps.crt", "rustfs-ca.crt")}
                    selfhost.setup(argparse.Namespace(**{key: None for key in vars(args)}))
                    self.assertEqual(originals, {name: (config / name).read_bytes() for name in originals})

    def test_control_host_inside_wildcard_rejected(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(selfhost, "CONFIG", Path(directory) / "local"):
            with self.assertRaisesRegex(ValueError, "inside the application wildcard"):
                selfhost.setup(self.args("control.example.com", "example.com"))


if __name__ == "__main__":
    unittest.main()
