import json
from pathlib import Path
import subprocess

# Leave headroom below WeChat's 200 MB single-download limit.
ALBUM_MAX_BYTES = 190 * 1024 * 1024


def format_selector(mode, max_bytes, direct=False):
    if mode == 'original':
        return 'bv*+ba/b'
    cap = min(max_bytes, ALBUM_MAX_BYTES)
    video = f"[vcodec~='^(avc1|h264)'][ext=mp4][filesize<=?{cap}]"
    compatible = f'bv{video}+ba[ext=m4a]/b{video}'
    if direct:
        compatible = f'b[ext=mp4][filesize<=?{cap}]'
    return compatible + f'/bv*[filesize<=?{max_bytes}]+ba/b[filesize<=?{max_bytes}]'


def probe(file):
    result = subprocess.run(['ffprobe', '-v', 'error', '-show_streams', '-show_format', '-of', 'json', str(file)],
                            check=True, capture_output=True, text=True, timeout=30)
    media = json.loads(result.stdout)
    streams = media.get('streams', [])
    video = next((s for s in streams if s.get('codec_type') == 'video'), None)
    audio = next((s for s in streams if s.get('codec_type') == 'audio'), None)
    if not video:
        raise ValueError('No video stream')
    duration = float(media.get('format', {}).get('duration') or video.get('duration') or 0)
    if duration <= 0:
        raise ValueError('Invalid video duration')
    compatible = Path(file).suffix.lower() == '.mp4' and video.get('codec_name') == 'h264' and (audio is None or audio.get('codec_name') == 'aac')
    return {'duration': duration, 'compatible': compatible}


def album_copy(file, max_bytes, duration):
    source = Path(file)
    cap = min(max_bytes, ALBUM_MAX_BYTES)
    video_kbps = min(4000, int(cap * 8 * 0.92 / duration / 1000) - 96)
    if video_kbps < 96:
        raise ValueError('Video duration or file size limit exceeded')
    target = source.with_name('compatible.mp4')
    subprocess.run([
        'ffmpeg', '-nostdin', '-hide_banner', '-loglevel', 'error', '-y', '-i', str(source),
        '-map', '0:v:0', '-map', '0:a:0?', '-sn', '-dn',
        '-vf', "scale=w='min(1280,iw)':h='min(720,ih)':force_original_aspect_ratio=decrease:force_divisible_by=2",
        '-c:v', 'libx264', '-preset', 'veryfast', '-threads', '1', '-pix_fmt', 'yuv420p',
        '-b:v', f'{video_kbps}k', '-maxrate', f'{video_kbps}k', '-bufsize', f'{video_kbps * 2}k',
        '-c:a', 'aac', '-b:a', '96k', '-movflags', '+faststart', '-map_metadata', '-1', str(target),
    ], check=True, capture_output=True, timeout=1800)
    metadata = probe(target)
    if target.stat().st_size > cap or not metadata['compatible'] or abs(metadata['duration'] - duration) > max(2, duration * 0.01):
        raise ValueError('Compatible video size or duration verification failed')
    return target, metadata
