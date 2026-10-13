import importlib.util
import json
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "deploy-cloud-run.py"
spec = importlib.util.spec_from_file_location("cloudrun_deploy", SCRIPT)
deploy = importlib.util.module_from_spec(spec)
spec.loader.exec_module(deploy)
EXAMPLE = Path(__file__).resolve().parents[1] / "infra" / "cloudrun.example.json"


class ConfigTests(unittest.TestCase):
    def config(self, edit=None):
        data = json.loads(EXAMPLE.read_text())
        if edit:
            edit(data)
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "config.json"
            path.write_text(json.dumps(data))
            return deploy.load_config(path)

    def test_example_is_valid_and_secrets_are_references(self):
        config = self.config()
        self.assertEqual(config["service"], "cellapp")
        flags = deploy.settings(config)
        self.assertIn("--set-secrets=", " ".join(flags))
        self.assertNotIn("replace-with-a-random-secret", " ".join(flags))
        self.assertIn("POSTGREST_URL=http://127.0.0.1:3001", " ".join(flags))
        self.assertIn("PGRST_SERVER_HOST=127.0.0.1", " ".join(flags))

    def test_app_data_jwt_must_be_shared_with_postgrest(self):
        with self.assertRaises(deploy.DeploymentError):
            self.config(lambda c: c["secrets"].update(PGRST_JWT_SECRET="different-version:1"))

    def test_rejects_overlap_or_plaintext_secret(self):
        for edit in (
            lambda c: c.update(apps_domain="example.com"),
            lambda c: c["secrets"].update(SECRET="plain-password"),
            lambda c: c.update(control_origin="http://control.example.com"),
        ):
            with self.subTest(edit=edit), self.assertRaises(deploy.DeploymentError):
                self.config(edit)

    def test_existing_secret_name_may_contain_uppercase_client_id(self):
        config = self.config(lambda c: c["secrets"].update(
            GITHUB_CLIENT_SECRET="cellapp-github-oauth-Ov23limVEZjXdRicZdzw:1"))
        self.assertEqual(config["secrets"]["GITHUB_CLIENT_SECRET"],
                         "cellapp-github-oauth-Ov23limVEZjXdRicZdzw:1")

    def test_preflight_checks_dependencies_and_scheduler_target(self):
        calls = []
        config = self.config()

        def fake_gcloud(stage, *args, **kwargs):
            calls.append(stage)
            if stage == "VPC subnet":
                return {"network": f"https://www.googleapis.com/compute/v1/projects/{config['project']}/global/networks/{config['network']}"}
            if stage == "cleanup scheduler":
                return {"state": "ENABLED", "httpTarget": {"uri": f"https://run.googleapis.com/v2/projects/{config['project']}/locations/{config['region']}/jobs/{config['service']}-cleanup:run"}}
            return {}

        with patch.object(deploy, "gcloud", side_effect=fake_gcloud):
            deploy.preflight(config)
        self.assertIn("VPC network", calls)
        self.assertIn("VPC subnet", calls)
        self.assertNotIn("VPC connector", calls)
        self.assertNotIn("Cloud SQL instance", calls)
        self.assertEqual(sum(stage.startswith("secret reference") for stage in calls), len(deploy.SECRET_KEYS))

        with patch.object(deploy, "gcloud", return_value={"state": "ENABLED", "httpTarget": {"uri": "https://example.com/wrong"}}):
            with self.assertRaises(deploy.DeploymentError):
                deploy.preflight(config)

    def test_missing_scheduler_blocks_release_preflight(self):
        config = self.config()

        def fake_gcloud(stage, *args, **kwargs):
            if stage == "cleanup scheduler":
                raise deploy.DeploymentError("cleanup scheduler missing")
            return {}

        with patch.object(deploy, "gcloud", side_effect=fake_gcloud):
            with self.assertRaises(deploy.DeploymentError):
                deploy.preflight(config)

    def test_direct_vpc_and_connector_are_exclusive(self):
        with self.assertRaises(deploy.DeploymentError):
            self.config(lambda c: c.update(vpc_connector="cellapp-connector"))
        connector = self.config(lambda c: (c.pop("network"), c.pop("subnet"), c.update(vpc_connector="cellapp-connector")))
        self.assertIn("--vpc-connector=cellapp-connector", deploy.settings(connector))
        direct = self.config()
        self.assertIn("--network=cellapp-test", deploy.settings(direct))
        self.assertIn("--subnet=cellapp-test-us-west1", deploy.settings(direct))

    def test_old_gcloud_jobs_capability_has_actionable_error(self):
        failed = subprocess.CompletedProcess([], 1, "", "Invalid choice: jobs")
        with patch.object(deploy.subprocess, "run", return_value=failed):
            with self.assertRaises(deploy.DeploymentError) as error:
                deploy.gcloud("Cloud Run Jobs capability", "run", "jobs", "deploy", "--help")
        self.assertIn("update gcloud", str(error.exception))

    def test_gcloud_failure_does_not_expose_output(self):
        failed = subprocess.CompletedProcess([], 1, "sensitive stdout", "sensitive stderr")
        with patch.object(deploy.subprocess, "run", return_value=failed):
            with self.assertRaises(deploy.DeploymentError) as error:
                deploy.gcloud("migration execution", "run", "jobs", "execute")
        self.assertNotIn("sensitive", str(error.exception))


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.config = json.loads(EXAMPLE.read_text())
        self.config["source_revision"] = "abc123def456"
        self.image = f"{self.config['region']}-docker.pkg.dev/{self.config['project']}/{self.config['repository']}/cellapp@sha256:" + "a" * 64

    def test_build_digest_is_used_for_jobs(self):
        commands = []
        self.config["source_revision"] = "abc123def456-20261008000000"

        def fake_gcloud(stage, *args, **kwargs):
            commands.append((stage, args))
            if stage == "image lookup":
                return {"image_summary": {"digest": "sha256:" + "a" * 64}}
            return {}

        with patch.object(deploy, "gcloud", side_effect=fake_gcloud):
            image = deploy.image_digest(self.config)
            name = deploy.deploy_job(self.config, image, "-migrate", "-migrate")
        self.assertEqual(name, "cellapp-migrate")
        self.assertEqual(image, self.image)
        self.assertIn("--ignore-file=.gcloudignore", dict(commands)["Cloud Build"])
        self.assertIn("--image=" + self.image, dict(commands)["cellapp-migrate job deploy"])

    def test_unready_revision_keeps_previous_traffic(self):
        calls = []

        def fake_gcloud(stage, *args, **kwargs):
            calls.append(stage)
            if stage == "service inventory":
                return [{"metadata": {"name": self.config["service"]}}]
            if stage == "migration execution lookup":
                return {"status": {"latestCreatedExecution": {"name": "cellapp-migrate-00001"}}}
            if stage == "migration result":
                return {"status": {"succeededCount": 1, "failedCount": 0, "completionTime": "2026-10-08T00:00:00Z"}}
            if stage == "service deploy":
                return {"status": {"latestCreatedRevisionName": "cellapp-00002"}}
            if stage == "revision readiness":
                return {"spec": {"containers": [{"image": self.image}]}, "status": {"conditions": [{"type": "Ready", "status": "False"}]}}
            return {}

        git = [subprocess.CompletedProcess([], 0, "", ""), subprocess.CompletedProcess([], 0, "abc123def456\n", "")]
        with patch.object(deploy, "preflight"), patch.object(deploy, "image_digest", return_value=self.image), patch.object(deploy.subprocess, "run", side_effect=git), patch.object(deploy, "gcloud", side_effect=fake_gcloud):
            with self.assertRaises(deploy.DeploymentError):
                deploy.release(self.config)
        self.assertNotIn("traffic switch", calls)

    def test_migration_failure_prevents_service_deploy(self):
        calls = []

        def fake_gcloud(stage, *args, **kwargs):
            calls.append(stage)
            if stage == "migration execution":
                raise deploy.DeploymentError("migration execution failed")
            return {}

        git = [subprocess.CompletedProcess([], 0, "", ""), subprocess.CompletedProcess([], 0, "abc123def456\n", "")]
        with patch.object(deploy, "preflight"), patch.object(deploy, "image_digest", return_value=self.image), patch.object(deploy.subprocess, "run", side_effect=git), patch.object(deploy, "gcloud", side_effect=fake_gcloud):
            with self.assertRaises(deploy.DeploymentError):
                deploy.release(self.config)
        self.assertNotIn("service deploy", calls)
        self.assertNotIn("cellapp-cleanup job deploy", calls)

    def test_unsuccessful_migration_result_prevents_service_deploy(self):
        calls = []

        def fake_gcloud(stage, *args, **kwargs):
            calls.append(stage)
            if stage == "migration execution lookup":
                return {"status": {"latestCreatedExecution": {"name": "cellapp-migrate-00001"}}}
            if stage == "migration result":
                return {"status": {"succeededCount": 0, "failedCount": 1, "completionTime": "2026-10-08T00:00:00Z"}}
            return {}

        git = [subprocess.CompletedProcess([], 0, "", ""), subprocess.CompletedProcess([], 0, "abc123def456\n", "")]
        with patch.object(deploy, "preflight"), patch.object(deploy, "image_digest", return_value=self.image), patch.object(deploy.subprocess, "run", side_effect=git), patch.object(deploy, "gcloud", side_effect=fake_gcloud):
            with self.assertRaises(deploy.DeploymentError):
                deploy.release(self.config)
        self.assertNotIn("service deploy", calls)

    def test_success_uses_same_digest_and_switches_after_ready(self):
        calls = []

        def fake_gcloud(stage, *args, **kwargs):
            calls.append((stage, args))
            if stage == "service inventory":
                return [{"metadata": {"name": self.config["service"]}}]
            if stage == "migration execution lookup":
                return {"status": {"latestCreatedExecution": {"name": "cellapp-migrate-00001"}}}
            if stage == "migration result":
                return {"status": {"succeededCount": 1, "failedCount": 0, "completionTime": "2026-10-08T00:00:00Z"}}
            if stage == "service deploy":
                return {"status": {"latestCreatedRevisionName": "cellapp-00001"}}
            if stage == "revision readiness":
                return {"spec": {"containers": [{"image": self.image}]}, "status": {"conditions": [{"type": "Ready", "status": "True"}]}}
            return {}

        git = [subprocess.CompletedProcess([], 0, "", ""), subprocess.CompletedProcess([], 0, "abc123def456\n", "")]
        with patch.object(deploy, "preflight"), patch.object(deploy, "image_digest", return_value=self.image), patch.object(deploy.subprocess, "run", side_effect=git), patch.object(deploy, "gcloud", side_effect=fake_gcloud), patch.object(deploy, "verify_routes"):
            deploy.release(self.config)
        stages = [stage for stage, _ in calls]
        self.assertLess(stages.index("migration execution"), stages.index("service deploy"))
        self.assertLess(stages.index("revision readiness"), stages.index("traffic switch"))
        self.assertTrue(any(self.image in arg for stage, args in calls for arg in args if stage.endswith("job deploy") or stage == "service deploy"))
        self.assertIn("--no-traffic", dict(calls)["service deploy"])

    def test_first_service_deploy_omits_no_traffic(self):
        calls = []

        def fake_gcloud(stage, *args, **kwargs):
            calls.append((stage, args))
            if stage == "service inventory":
                return []
            if stage == "migration execution lookup":
                return {"status": {"latestCreatedExecution": {"name": "cellapp-migrate-00001"}}}
            if stage == "migration result":
                return {"status": {"succeededCount": 1, "failedCount": 0, "completionTime": "2026-10-08T00:00:00Z"}}
            if stage == "service deploy":
                return {"status": {"latestCreatedRevisionName": "cellapp-00001"}}
            if stage == "revision readiness":
                return {"spec": {"containers": [{"image": self.image}]},
                        "status": {"conditions": [{"type": "Ready", "status": "True"}]}}
            return {}

        git = [subprocess.CompletedProcess([], 0, "", ""), subprocess.CompletedProcess([], 0, "abc123def456\n", "")]
        with patch.object(deploy, "preflight"), patch.object(deploy, "image_digest", return_value=self.image), patch.object(deploy.subprocess, "run", side_effect=git), patch.object(deploy, "gcloud", side_effect=fake_gcloud), patch.object(deploy, "verify_routes"):
            deploy.release(self.config)
        self.assertNotIn("--no-traffic", dict(calls)["service deploy"])

    def test_route_failure_is_not_success(self):
        with patch.object(deploy, "probe", side_effect=[(200, "control-id"), (200, "wrong-route-id")]):
            with self.assertRaises(deploy.DeploymentError):
                deploy.verify_routes(self.config)


if __name__ == "__main__":
    unittest.main()
