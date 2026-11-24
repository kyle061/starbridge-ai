import importlib.util
from pathlib import Path
import tempfile
import types
import unittest
from unittest.mock import patch


SCRIPT = Path(__file__).parents[1] / "install_imagegen.py"
SPEC = importlib.util.spec_from_file_location("install_imagegen", SCRIPT)
installer = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(installer)


class InstallImagegenTests(unittest.TestCase):
    def test_installs_skill_without_changing_provider_and_runs_only_check(self):
        for windows in (False, True):
            with self.subTest(windows=windows), tempfile.TemporaryDirectory() as directory:
                home = Path(directory)
                codex_home = home / "custom-codex"
                codex_home.mkdir()
                config = codex_home / "config.toml"
                config.write_text('model_provider="custom"\n')
                runtime = home / ".cache/starbridge-imagegen/venv"
                python = runtime / ("Scripts/python.exe" if windows else "bin/python")
                python.parent.mkdir(parents=True)
                python.touch()
                with patch.object(installer.Path, "home", return_value=home), patch.object(installer.sys, "platform", "win32" if windows else "linux"), patch.object(installer.subprocess, "run", return_value=types.SimpleNamespace(returncode=0)) as run:
                    self.assertEqual(installer.install(codex_home), 0)
                destination = codex_home / "skills/starbridge-imagegen"
                for relative in ("SKILL.md", "scripts/with_current_provider.py"):
                    self.assertEqual((destination / relative).read_bytes(), (SCRIPT.parent / "starbridge-imagegen" / relative).read_bytes())
                self.assertEqual(config.read_text(), 'model_provider="custom"\n')
                self.assertEqual(run.call_args_list[0].args[0], [str(python), "-m", "pip", "install", "openai", "pillow"])
                self.assertEqual(run.call_args.args[0], [str(python), str(destination / "scripts/with_current_provider.py"), "--check"])
                self.assertEqual(run.call_args.kwargs["env"]["CODEX_HOME"], str(codex_home))


if __name__ == "__main__":
    unittest.main()
