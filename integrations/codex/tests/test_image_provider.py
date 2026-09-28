import importlib.util
import json
import os
from pathlib import Path
import tempfile
import unittest
import sys
import types
from unittest.mock import patch

SCRIPT = Path(__file__).parents[1] / "starbridge-imagegen/scripts/with_current_provider.py"
SPEC = importlib.util.spec_from_file_location("image_provider", SCRIPT)
launcher = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(launcher)


class ImageProviderTests(unittest.TestCase):
    def test_launcher_scopes_credentials_and_disables_sdk_retries(self):
        calls = []
        sdk = types.ModuleType("openai")
        original_client = lambda **kwargs: calls.append((kwargs, os.environ.get("OPENAI_API_KEY")))
        sdk.OpenAI = sdk.AsyncOpenAI = original_client

        def invoke(path, run_name):
            self.assertEqual(path, "/bundled/image_gen.py")
            self.assertEqual(run_name, "__main__")
            self.assertEqual(sys.argv[1:3], ["generate", "--dry-run"])
            sdk.OpenAI()
            sdk.AsyncOpenAI()
            self.assertEqual(os.environ["OPENAI_BASE_URL"], "https://starbridaeai.top/v1/")

        with patch.dict(sys.modules, {"openai": sdk}), patch.dict(os.environ, {"OPENAI_API_KEY": "parent"}), patch.object(launcher.runpy, "run_path", side_effect=invoke):
            launcher.run_bundled_cli(Path("/bundled/image_gen.py"), ["generate", "--dry-run"], {"OPENAI_API_KEY": "selected", "OPENAI_BASE_URL": "https://starbridaeai.top/v1/"})
            self.assertEqual(os.environ["OPENAI_API_KEY"], "parent")
            self.assertIs(sdk.OpenAI, original_client)
        self.assertEqual(calls, [({"max_retries": 0, "timeout": 300.0}, "selected")] * 2)

    def config(self, text):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        path = Path(directory.name) / "config.toml"
        path.write_text(text)
        return path

    def test_reuses_only_selected_provider_without_mutating_parent(self):
        path = self.config('''model_provider="starbridge"
[model_providers.starbridge]
base_url="https://starbridaeai.top/v1"
experimental_bearer_token="selected-private-key"
[model_providers.other]
base_url="https://unrelated.example/v1"
experimental_bearer_token="other-private-key"
''')
        parent = {"OPENAI_API_KEY": "unrelated-env-key", "OPENAI_ORG_ID": "other-org"}
        child, status = launcher.provider_environment(path, parent)
        self.assertEqual(child["OPENAI_API_KEY"], "selected-private-key")
        self.assertEqual(child["OPENAI_BASE_URL"], "https://starbridaeai.top/v1/")
        self.assertNotIn("OPENAI_ORG_ID", child)
        self.assertEqual(parent["OPENAI_API_KEY"], "unrelated-env-key")
        self.assertNotIn("private-key", json.dumps(status))

    def test_configured_env_key_and_profile(self):
        path = self.config('''model_provider="other"
[profiles.images]
model_provider="starbridge"
[model_providers.starbridge]
base_url="https://starbridaeai.top/openai/v1/"
env_key="STARBRIDGE_TEST_KEY"
experimental_bearer_token="fallback"
''')
        child, _ = launcher.provider_environment(path, {"STARBRIDGE_TEST_KEY": "selected"}, "images")
        self.assertEqual(child["OPENAI_API_KEY"], "selected")
        with self.assertRaisesRegex(ValueError, "profile"):
            launcher.provider_environment(path, {}, "missing")

    def test_rejects_other_providers_insecure_urls_and_missing_credentials(self):
        for endpoint in ("https://other.example/v1", "http://starbridaeai.top/v1", "https://starbridaeai.top.attacker.test/v1", "https://user@starbridaeai.top/v1", "https://starbridaeai.top/v1?key=private"):
            with self.subTest(endpoint=endpoint):
                path = self.config(f'model_provider="test"\n[model_providers.test]\nbase_url="{endpoint}"\nexperimental_bearer_token="private"\n')
                with self.assertRaisesRegex(ValueError, "no credentials were used"):
                    launcher.provider_environment(path, {})
        path = self.config('model_provider="test"\n[model_providers.test]\nbase_url="https://starbridaeai.top/v1"\n')
        with self.assertRaisesRegex(ValueError, "no configured credential"):
            launcher.provider_environment(path, {"OPENAI_API_KEY": "unrelated-secret"})


if __name__ == "__main__":
    unittest.main()
