"""Download original image bytes with the resolver's guarded, cookie-aware transport."""
from pathlib import Path
import urllib.parse
import zipfile
from yt_dlp.networking import Request

MAX_IMAGES = 100
IMAGE_MAX_BYTES = 30 * 1024 * 1024


def image_ext(data):
    if data.startswith(b'\xff\xd8\xff'):
        return 'jpg'
    if data.startswith(b'\x89PNG\r\n\x1a\n'):
        return 'png'
    if data[:6] in (b'GIF87a', b'GIF89a'):
        return 'gif'
    if data[:4] == b'RIFF' and data[8:12] == b'WEBP':
        return 'webp'
    if data[4:8] == b'ftyp' and data[8:12] in (b'avif', b'avis'):
        return 'avif'
    raise ValueError('Image source unavailable or invalid image format')


def safe_url(url):
    parsed = urllib.parse.urlsplit(url)
    if parsed.scheme != 'https' or not parsed.hostname or parsed.username or parsed.password or parsed.port not in (None, 443):
        raise ValueError('Unsupported image source URL')
    return url


def download_images(ydl, images, directory, max_bytes, headers, emit):
    if not images or len(images) > MAX_IMAGES:
        raise ValueError('Image count size limit exceeded')
    assets, total = [], 0
    for index, image in enumerate(images):
        result = None
        for url in image['urls'][:3]:
            target = Path(directory) / f'image-{index:03d}.part'
            try:
                with ydl.urlopen(Request(safe_url(url), headers=headers)) as response, target.open('wb') as output:
                    safe_url(response.url)
                    size = 0
                    while chunk := response.read(65536):
                        size += len(chunk)
                        if size > IMAGE_MAX_BYTES or total + size > max_bytes:
                            raise ValueError('Image file size limit exceeded')
                        output.write(chunk)
                with target.open('rb') as source:
                    ext = image_ext(source.read(32))
                if size < 32:
                    raise ValueError('Invalid image source')
                filename = f'image-{index:03d}.{ext}'
                target.rename(Path(directory) / filename)
                result = {'filename': filename, 'ext': ext, 'kind': 'image', 'compatible': ext in ('jpg', 'png', 'gif')}
                total += size
                break
            except Exception as error:
                target.unlink(missing_ok=True)
                if 'size limit' in str(error):
                    raise
        if not result:
            raise ValueError('Image source unavailable')
        assets.append(result)
        emit('progress', value=(index + 1) / len(images) * 95)
    archive = None
    if len(assets) > 1:
        archive = 'originals.zip'
        # Store without recompressing: downloaded image bytes stay exactly the same.
        with zipfile.ZipFile(Path(directory) / archive, 'w', compression=zipfile.ZIP_STORED) as bundle:
            for asset in assets:
                bundle.write(Path(directory) / asset['filename'], asset['filename'])
    return assets, archive
