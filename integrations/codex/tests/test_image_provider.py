import importlib.util
import itertools
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
            self.assertEqual(path, str(Path("/bundled/image_gen.py")))
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

    def test_selects_latest_authorized_version_numerically(self):
        cases = [
            (["gpt-image-2", "gpt-image-2.5-sunburst", "gpt-image-2.5-flare"], "gpt-image-2.5-flare"),
            (["gpt-image-2.9", "gpt-image-2.10-flare"], "gpt-image-2.10-flare"),
            (["gpt-image-2.5.9", "gpt-image-2.5.10"], "gpt-image-2.5.10"),
            (["gpt-image-2.5-flare", "gpt-image-3"], "gpt-image-3"),
            (["gpt-image-2", "gpt-image-2.0"], "gpt-image-2"),
            (["gpt-image-2", "gpt-image-latest", "gpt-image-99bad", "gpt-image-*", None], "gpt-image-2"),
            (["gpt-6-astra", "gpt-image-2"], "gpt-image-2"),
        ]
        for model_ids, expected in cases:
            with self.subTest(model_ids=model_ids):
                self.assertEqual(launcher.preferred_image_model(model_ids), expected)
                self.assertEqual(launcher.preferred_image_model(reversed(model_ids)), expected)
        for order in itertools.permutations(["gpt-image-2", "gpt-image-2.5-sunburst", "gpt-image-2.5-flare"]):
            self.assertEqual(launcher.preferred_image_model(order), "gpt-image-2.5-flare")

    def test_discovers_models_only_without_explicit_choice(self):
        for args, expected in [
            (["generate", "--prompt", "draw"], "gpt-image-2.10-flare"),
            (["edit", "--image", "input.png"], "gpt-image-2.10-flare"),
            (["generate-batch", "--input", "prompts.jsonl"], "gpt-image-2.10-flare"),
            (["generate", "--model", "gpt-image-2"], "gpt-image-2"),
            (["generate", "--model=gpt-image-2"], "gpt-image-2"),
        ]:
            with self.subTest(args=args):
                lookups, forwarded = [], []

                def list_models():
                    lookups.append(os.environ["OPENAI_API_KEY"])
                    return types.SimpleNamespace(data=[types.SimpleNamespace(id=model) for model in ["gpt-image-2", "gpt-image-2.9", "gpt-image-2.10-flare"]])

                sdk = types.ModuleType("openai")
                sdk.OpenAI = sdk.AsyncOpenAI = lambda **kwargs: types.SimpleNamespace(models=types.SimpleNamespace(list=list_models))
                with patch.dict(sys.modules, {"openai": sdk}), patch.object(launcher.runpy, "run_path", side_effect=lambda *a, **kw: forwarded.extend(sys.argv[1:])):
                    launcher.run_bundled_cli(Path("image_gen.py"), args, {"OPENAI_API_KEY": "selected"})
                if launcher.has_explicit_model(args):
                    self.assertEqual(lookups, [])
                    self.assertEqual(forwarded, args)
                else:
                    self.assertEqual(lookups, ["selected"])
                    self.assertEqual(forwarded[-2:], ["--model", expected])
                if args[0] == "generate-batch":
                    self.assertEqual(forwarded[forwarded.index("--max-attempts") + 1], "1")

    def test_no_image_models_stops_before_generation(self):
        sdk = types.ModuleType("openai")
        sdk.OpenAI = sdk.AsyncOpenAI = lambda **kwargs: types.SimpleNamespace(models=types.SimpleNamespace(list=lambda: types.SimpleNamespace(data=[])))
        with patch.dict(sys.modules, {"openai": sdk}), patch.object(launcher.runpy, "run_path") as invoke:
            with self.assertRaisesRegex(ValueError, "no available GPT image model"):
                launcher.run_bundled_cli(Path("image_gen.py"), ["generate"], {"OPENAI_API_KEY": "selected"})
            invoke.assert_not_called()

    def test_runtime_paths_support_windows_and_posix(self):
        home = Path("user")
        dedicated, bundled = launcher.runtime_paths(home, True)
        self.assertEqual(dedicated, home / ".cache/starbridge-imagegen/venv/Scripts/python.exe")
        self.assertEqual(bundled, home / ".cache/codex-runtimes/codex-primary-runtime/dependencies/python/python.exe")
        dedicated, bundled = launcher.runtime_paths(home, False)
        self.assertEqual(dedicated, home / ".cache/starbridge-imagegen/venv/bin/python")
        self.assertEqual(bundled.name, "python3")

    def test_uses_windows_dedicated_runtime(self):
        with tempfile.TemporaryDirectory() as directory:
            home = Path(directory)
            dedicated, _ = launcher.runtime_paths(home, True)
            dedicated.parent.mkdir(parents=True)
            dedicated.touch()
            with patch.object(launcher.Path, "home", return_value=home), patch.object(launcher.sys, "platform", "win32"), patch.object(launcher.os, "execv", side_effect=SystemExit) as execute:
                with self.assertRaises(SystemExit):
                    launcher.use_available_runtime()
                self.assertEqual(execute.call_args.args[0], str(dedicated))

    def test_check_reports_missing_dependencies_as_not_ready(self):
        path = self.config('model_provider="starbridge"\n[model_providers.starbridge]\nbase_url="https://starbridaeai.top/v1"\nexperimental_bearer_token="selected"\n')
        for missing in ("openai", "PIL"):
            with self.subTest(missing=missing), patch.object(launcher, "use_available_runtime"), patch.object(launcher.sys, "argv", [str(SCRIPT), "--config", str(path), "--check"]), patch.object(launcher.Path, "is_file", return_value=True), patch.object(launcher.importlib.util, "find_spec", side_effect=lambda name: None if name == missing else object()), patch("builtins.print") as output:
                self.assertEqual(launcher.main(), 2)
                status = json.loads(output.call_args.args[0])
                self.assertFalse(status["ready"])
                self.assertNotIn("selected", output.call_args.args[0])

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
