import importlib.util
from pathlib import Path
import stat
import tempfile
import unittest

DIRECTORY = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location("init_env", DIRECTORY / "init-env.py")
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class InitEnvTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name)
        (self.directory / ".env.example").write_text((DIRECTORY / ".env.example").read_text())

    def test_secrets_are_private_distinct_and_never_overwritten(self):
        path = MODULE.create_env(self.directory, "owner@example.com")
        original = path.read_text()
        values = dict(line.split("=", 1) for line in original.splitlines() if "=" in line and not line.startswith("#"))
        secret_keys = ["ADMIN_PASSWORD", "POSTGRES_PASSWORD", "REDIS_PASSWORD", "JWT_SECRET", "TOTP_ENCRYPTION_KEY"]
        self.assertEqual(len({values[k] for k in secret_keys}), 5)
        for key in secret_keys:
            self.assertRegex(values[key], r"^[a-f0-9]{64}$")
        self.assertEqual(stat.S_IMODE(path.stat().st_mode), 0o600)
        with self.assertRaises(FileExistsError):
            MODULE.create_env(self.directory, "other@example.com")
        self.assertEqual(path.read_text(), original)

    def test_email_cannot_inject_compose_values(self):
        for value in ["bad", "a@example.com\nBIND_HOST=0.0.0.0", "${SECRET}@example.com"]:
            with self.assertRaises(ValueError):
                MODULE.create_env(self.directory, value)
        self.assertFalse((self.directory / ".env").exists())


if __name__ == "__main__":
    unittest.main()
