#!/usr/bin/python3 -I
"""Forced SSH command for the unprivileged starbridge-deploy account."""
import os
from pathlib import Path
import re
import shlex
import subprocess
import sys
import tempfile

INCOMING = Path('/var/lib/starbridge-deploy/incoming')
MAX_ARCHIVE_BYTES = 1024 * 1024 * 1024


def parse_command(command):
    args = shlex.split(command)
    if len(args) < 3 or args[0] != 'starbridge':
        raise ValueError('Only Starbridge deployment commands are permitted.')
    mode = args[1]
    values = args[2:]
    if mode == 'upload' and len(values) == 1:
        if re.fullmatch('[a-f0-9]{40}', values[0]):
            return mode, values
    if mode in ('prepare', 'apply') and len(values) == (1 if mode == 'prepare' else 3):
        if not re.fullmatch('[1-9][0-9]{0,4}', values[0]) or not 1024 <= int(values[0]) <= 65535:
            raise ValueError('Invalid application port.')
        if mode == 'apply' and (
            not re.fullmatch(r'[A-Za-z0-9._+%-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}', values[1])
            or not re.fullmatch('[a-f0-9]{40}', values[2])
        ):
            raise ValueError('Invalid email or image revision.')
        return mode, values
    raise ValueError('Command is not permitted.')


def main():
    mode, values = parse_command(os.environ.get('SSH_ORIGINAL_COMMAND', ''))
    if mode == 'upload':
        temporary = None
        try:
            with tempfile.NamedTemporaryFile(dir=str(INCOMING), prefix='.upload-', delete=False) as output:
                temporary = output.name
                size = 0
                while True:
                    chunk = sys.stdin.buffer.read(1024 * 1024)
                    if not chunk:
                        break
                    size += len(chunk)
                    if size > MAX_ARCHIVE_BYTES:
                        raise ValueError('Image archive exceeds the 1 GiB limit.')
                    output.write(chunk)
            os.replace(temporary, str(INCOMING / 'image.tar.gz'))
            temporary = None
            print('Starbridge image uploaded.')
        finally:
            if temporary:
                os.unlink(temporary)
        return 0
    return subprocess.call(['/usr/bin/sudo', '-n', '/usr/local/sbin/starbridge-deploy-admin', mode] + values)


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (ValueError, OSError) as error:
        sys.exit(str(error))
