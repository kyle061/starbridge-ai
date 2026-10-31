"""Create album copies on request; never rewrite the downloaded originals."""
import argparse
import json
from pathlib import Path
import subprocess
from media_compat import ALBUM_MAX_BYTES, album_copy, probe


def prepare(files, max_bytes):
    assets = []
    for index, asset in enumerate(files):
        file = Path(asset['file'])
        if asset['compatible'] and file.stat().st_size <= ALBUM_MAX_BYTES:
            target = file
        elif asset['kind'] == 'video':
            target, _ = album_copy(file, max_bytes, probe(file)['duration'])
        else:
            target = file.with_name(f'album-{index:03d}.jpg')
            subprocess.run(['ffmpeg', '-nostdin', '-v', 'error', '-y', '-i', str(file),
                            '-frames:v', '1', '-threads', '1', '-q:v', '2', str(target)],
                           check=True, capture_output=True, timeout=60)
            if target.stat().st_size > ALBUM_MAX_BYTES:
                raise ValueError('Image size limit')
        assets.append({'filename': target.name})
    return assets


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--files', required=True)
    parser.add_argument('--max-mb', type=int, required=True)
    args = parser.parse_args()
    try:
        assets = prepare(json.loads(args.files), args.max_mb * 1048576)
        print(json.dumps({'type': 'complete', 'assets': assets}), flush=True)
    except Exception:
        print(json.dumps({'type': 'error', 'code': 'ALBUM_FAILED', 'message': '相册副本生成失败，原文件仍可保存，请稍后重试'}), flush=True)
        return 1
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
