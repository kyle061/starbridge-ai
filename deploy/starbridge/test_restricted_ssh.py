import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest

SOURCE = Path(__file__).resolve().parent


def load(name):
    spec = importlib.util.spec_from_file_location(name, SOURCE / (name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


gateway = load('ssh-gateway')
admin = load('deploy-admin')
REVISION = 'a' * 40


class RestrictedSSHTests(unittest.TestCase):
    def test_allows_only_expected_commands(self):
        for command in ['starbridge prepare 18080', 'starbridge upload ' + REVISION,
                        'starbridge apply 18080 admin@example.com ' + REVISION]:
            self.assertTrue(gateway.parse_command(command))

    def test_rejects_shell_commands_and_injection(self):
        for command in ['', 'id', 'bash -s', 'scp -t /root', 'starbridge prepare 80',
                        'starbridge prepare 18080;id', 'starbridge prepare 18080 extra',
                        'starbridge upload ../../image',
                        'starbridge apply 18080 $(id) ' + REVISION,
                        'starbridge apply 18080 admin@example.com latest']:
            with self.subTest(command=command), self.assertRaises(ValueError):
                gateway.parse_command(command)

    def check_manifest(self, manifest, repositories=None):
        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary) / 'image.tar.gz'
            with tarfile.open(path, 'w:gz') as archive:
                entries = {'manifest.json': manifest}
                if repositories is not None:
                    entries['repositories'] = repositories
                for name, content in entries.items():
                    data = json.dumps(content).encode()
                    member = tarfile.TarInfo(name)
                    member.size = len(data)
                    archive.addfile(member, io.BytesIO(data))
            admin.validate_archive(path, REVISION)

    def test_accepts_single_revision_image(self):
        self.check_manifest([{'RepoTags': ['starbridge-ai:' + REVISION]}])
        self.check_manifest([{'RepoTags': ['starbridge-ai:' + REVISION]}], {'starbridge-ai': {REVISION: 'config'}})

    def test_rejects_images_that_would_overwrite_other_projects(self):
        for manifest in [[], [{'RepoTags': ['eio-api:latest']}],
                         [{'RepoTags': ['starbridge-ai:' + REVISION, 'eio-api:latest']}],
                         [{'RepoTags': ['starbridge-ai:' + REVISION]}, {'RepoTags': ['other:latest']}]]:
            with self.subTest(manifest=manifest), self.assertRaises(ValueError):
                self.check_manifest(manifest)
        with self.assertRaises(ValueError):
            self.check_manifest([{'RepoTags': ['starbridge-ai:' + REVISION]}], {'eio-api': {'latest': 'config'}})


if __name__ == '__main__':
    unittest.main()
