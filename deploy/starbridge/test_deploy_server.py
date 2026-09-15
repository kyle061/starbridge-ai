"""Exercise deployment isolation with a fake Docker daemon and real local files/sockets."""
import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import sys
import tempfile
import unittest

SOURCE = Path(__file__).resolve().parent
IMAGE = "starbridge-ai:" + "a" * 40
MARKER = "https://github.com/kyle061/starbridge-ai\n"


class DeployServerTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.base = Path(self.temporary.name).resolve()
        self.root = self.base / "starbridge"
        self.bin = self.base / "bin"
        self.bin.mkdir()
        self.log = self.base / "docker.jsonl"
        self.env = dict(os.environ, PATH=f"{self.bin}:{os.environ['PATH']}", DOCKER_LOG=str(self.log))
        self.env.pop("FAKE_CONTAINERS", None)
        self.env.pop("FAKE_VOLUME", None)
        self.env.pop("FAIL_UP", None)
        docker = self.bin / "docker"
        docker.write_text(f"#!{sys.executable}\n" + '''import json, os, sys
args = sys.argv[1:]
with open(os.environ["DOCKER_LOG"], "a") as f:
    f.write(json.dumps(args) + "\\n")
containers = json.loads(os.environ.get("FAKE_CONTAINERS", "[]"))
if args[0] == "ps":
    print("\\n".join(c["Id"] for c in containers))
elif args[0] == "inspect":
    print(json.dumps(containers))
elif args[:2] == ["volume", "ls"]:
    print(os.environ.get("FAKE_VOLUME", ""))
elif args[0] == "compose" and "up" in args and os.environ.get("FAIL_UP"):
    sys.exit(1)
''')
        docker.chmod(0o755)
        curl = self.bin / "curl"
        curl.write_text("#!/bin/sh\nexit 0\n")
        curl.chmod(0o755)

    def port(self):
        with socket.socket() as sock:
            sock.bind(("0.0.0.0", 0))
            return sock.getsockname()[1]

    def run_script(self, mode="prepare", port=None, root=None):
        return subprocess.run(
            ["bash", str(SOURCE / "deploy-server.sh"), mode, str(root or self.root),
             str(port or self.port()), "admin@example.com", IMAGE],
            env=self.env, capture_output=True, text=True,
        )

    def managed(self):
        self.root.mkdir()
        (self.root / ".starbridge-managed").write_text(MARKER)

    def commands(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()]

    def test_existing_directory_is_not_adopted_or_modified(self):
        self.root.mkdir()
        important = self.root / "existing-project.txt"
        important.write_text("keep this")
        result = self.run_script()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("already contains files", result.stderr)
        self.assertEqual(important.read_text(), "keep this")
        self.assertEqual(list(self.root.iterdir()), [important])

    def test_unsafe_directory_rejected_before_docker(self):
        result = self.run_script(root=str(self.base) + "/../starbridge")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.log.exists())

    def test_existing_volume_blocks_new_installation(self):
        self.env["FAKE_VOLUME"] = "starbridge_postgres_data"
        result = self.run_script()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Existing starbridge volumes", result.stderr)
        self.assertFalse(self.root.exists())

    def test_busy_port_does_not_stop_service(self):
        with socket.socket() as listener:
            listener.bind(("0.0.0.0", 0))
            listener.listen()
            result = self.run_script(port=listener.getsockname()[1])
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("already occupied", result.stderr)
            self.assertFalse(any("up" in cmd or "stop" in cmd or "down" in cmd for cmd in self.commands()))

    def test_foreign_compose_directory_blocks_update(self):
        self.managed()
        self.env["FAKE_CONTAINERS"] = json.dumps([{
            "Id": "foreign", "Config": {"Labels": {"com.docker.compose.project.working_dir": "/other/project"}}
        }])
        result = self.run_script()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("another directory", result.stderr)

    def test_owned_gateway_may_keep_its_existing_port(self):
        self.managed()
        with socket.socket() as listener:
            listener.bind(("0.0.0.0", 0))
            listener.listen()
            port = listener.getsockname()[1]
            self.env["FAKE_CONTAINERS"] = json.dumps([{
                "Id": "own", "Config": {"Labels": {
                    "com.docker.compose.project.working_dir": str(self.root / "deploy/starbridge"),
                    "com.docker.compose.service": "gateway",
                }}, "State": {"Running": True},
                "NetworkSettings": {"Ports": {"8080/tcp": [{"HostPort": str(port)}]}},
            }])
            result = self.run_script(port=port)
            self.assertEqual(result.returncode, 0, result.stderr)

    def stage_apply(self):
        result = self.run_script()
        self.assertEqual(result.returncode, 0, result.stderr)
        directory = self.root / "deploy/starbridge"
        for name in [".env.example", "init-env.py", "compose.yaml"]:
            shutil.copyfile(SOURCE / name, directory / name)
        (directory / ".env").write_text("# custom settings\nADMIN_PASSWORD=keep-me\nJWT_SECRET=keep-jwt\nUPSTREAM_HOSTS=custom.example\nAPP_PORT=9999\n")
        (self.root / "starbridge-image.tar.gz").write_bytes(b"mock image")
        return directory

    def test_apply_preserves_secrets_and_starts_only_named_project(self):
        directory = self.stage_apply()
        chosen_port = self.port()
        result = self.run_script(mode="apply", port=chosen_port)
        self.assertEqual(result.returncode, 0, result.stderr)
        contents = (directory / ".env").read_text()
        for value in ["# custom settings", "ADMIN_PASSWORD=keep-me", "JWT_SECRET=keep-jwt",
                      "UPSTREAM_HOSTS=custom.example", f"APP_PORT={chosen_port}", f"STARBRIDGE_IMAGE={IMAGE}"]:
            self.assertIn(value, contents)
        self.assertEqual((directory / ".env").stat().st_mode & 0o777, 0o600)
        starts = [cmd for cmd in self.commands() if "up" in cmd]
        self.assertEqual(len(starts), 1)
        self.assertEqual(starts[0][:3], ["compose", "-p", "starbridge"])
        self.assertIn("--no-build", starts[0])
        self.assertIn("--wait", starts[0])
        self.assertEqual(starts[0][-3:], ["gateway", "postgres", "redis"])
        self.assertFalse((self.root / "starbridge-image.tar.gz").exists())

    def test_unhealthy_update_is_reported_as_failure(self):
        self.stage_apply()
        self.env["FAIL_UP"] = "1"
        result = self.run_script(mode="apply")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("did not become healthy", result.stderr)
        self.assertTrue((self.root / "starbridge-image.tar.gz").exists())
        self.assertFalse(any("down" in cmd or "prune" in cmd for cmd in self.commands()))


if __name__ == "__main__":
    unittest.main()
