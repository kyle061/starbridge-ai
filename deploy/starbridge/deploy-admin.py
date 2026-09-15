#!/usr/bin/python3 -I
"""Root-owned helper: accepts only fixed Starbridge operations and image tags."""
import fcntl
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys
import tarfile

ROOT = Path('/opt/starbridge')
INCOMING = Path('/var/lib/starbridge-deploy/incoming/image.tar.gz')


def validate_archive(path, revision):
    with tarfile.open(str(path), 'r:gz') as archive:
        member = archive.getmember('manifest.json')
        if not member.isfile() or member.size > 1024 * 1024:
            raise ValueError('Invalid image manifest.')
        manifest = json.load(archive.extractfile(member))
        expected = ['starbridge-ai:' + revision]
        if not isinstance(manifest, list) or len(manifest) != 1 or manifest[0].get('RepoTags') != expected:
            raise ValueError('Archive must contain only the requested Starbridge image tag.')
        # Legacy repositories metadata must not retag unrelated images during docker load.
        if 'repositories' in archive.getnames():
            member = archive.getmember('repositories')
            if not member.isfile() or member.size > 1024 * 1024:
                raise ValueError('Invalid repositories metadata.')
            repositories = json.load(archive.extractfile(member))
            if set(repositories) != {'starbridge-ai'} or set(repositories['starbridge-ai']) != {revision}:
                raise ValueError('Repositories metadata must reference only this Starbridge revision.')


def main():
    args = sys.argv[1:]
    if len(args) not in (2, 4) or args[0] not in ('prepare', 'apply'):
        raise ValueError('Only prepare PORT or apply PORT EMAIL REVISION is supported.')
    mode, port = args[:2]
    if len(args) != (2 if mode == 'prepare' else 4):
        raise ValueError('Unexpected arguments.')
    if not re.fullmatch('[1-9][0-9]{0,4}', port) or not 1024 <= int(port) <= 65535:
        raise ValueError('Invalid application port.')
    command = ['/bin/bash', str(ROOT / 'deploy/starbridge/deploy-server.sh'), mode, str(ROOT), port]
    os.environ.clear()
    os.environ.update(PATH='/usr/sbin:/usr/bin:/sbin:/bin', HOME='/root', LC_ALL='C')
    if mode == 'apply':
        email, revision = args[2:]
        if not re.fullmatch(r'[A-Za-z0-9._+%-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}', email):
            raise ValueError('Invalid administrator email.')
        if not re.fullmatch('[a-f0-9]{40}', revision):
            raise ValueError('Invalid image revision.')
        # Snapshot to a root-owned file so the uploading account cannot change it after validation.
        fd = os.open(str(INCOMING), os.O_RDONLY | os.O_NOFOLLOW)
        with os.fdopen(fd, 'rb') as source:
            info = os.fstat(source.fileno())
            if not stat.S_ISREG(info.st_mode) or info.st_size > 1024 * 1024 * 1024:
                raise ValueError('Invalid image archive.')
            archive = ROOT / 'starbridge-image.tar.gz'
            with archive.open('wb') as output:
                shutil.copyfileobj(source, output)
        validate_archive(archive, revision)
        command += [email, 'starbridge-ai:' + revision]
    result = subprocess.call(command)
    if mode == 'apply' and result == 0:
        INCOMING.unlink()
    return result


if __name__ == '__main__':
    try:
        with open('/run/lock/starbridge-deploy.lock', 'w') as lock:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            sys.exit(main())
    except (ValueError, OSError, KeyError, tarfile.TarError) as error:
        sys.exit(str(error))
