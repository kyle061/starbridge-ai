import io
from pathlib import Path
import sys
import tempfile
import unittest
import zipfile
import contextlib
import json
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parents[1] / 'worker'))
from images import download_images, image_ext, safe_url, MAX_IMAGES


class Response(io.BytesIO):
    url = 'https://cdn.example.org/image.jpg'


class Transport:
    def __init__(self, data):
        self.data = data
        self.calls = []

    def urlopen(self, request):
        self.calls.append(request)
        return Response(self.data)


class ImagesTests(unittest.TestCase):
    def test_original_bytes_order_and_zip_are_preserved(self):
        data = (Path(__file__).parents[1] / 'web/assets/demo.jpg').read_bytes()
        transport = Transport(data)
        with tempfile.TemporaryDirectory() as tmp:
            assets, archive = download_images(transport, [{'urls': [Response.url]}] * 2, Path(tmp),
                                               len(data) * 2, {'Referer': 'https://www.douyin.com/'}, lambda *a, **k: None)
            self.assertEqual(len(assets), 2)
            with zipfile.ZipFile(Path(tmp) / archive) as bundle:
                self.assertEqual(bundle.namelist(), ['image-000.jpg', 'image-001.jpg'])
                for asset in assets:
                    self.assertEqual((Path(tmp) / asset['filename']).read_bytes(), data)
                    self.assertEqual(bundle.read(asset['filename']), data)
            self.assertEqual(transport.calls[0].headers['Referer'], 'https://www.douyin.com/')

    def test_resolver_defaults_to_original_and_emits_image_manifest(self):
        import resolve
        data = (Path(__file__).parents[1] / 'web/assets/demo.jpg').read_bytes()
        transport = Transport(data)
        class Ydl(Transport):
            def __enter__(self): return self
            def __exit__(self, *args): pass
        with tempfile.TemporaryDirectory() as tmp:
            argv = ['resolve', '--url', 'https://cdn.example.org/image.jpg', '--platform', 'direct',
                    '--directory', tmp, '--max-mb', '1', '--max-duration', '60']
            output = io.StringIO()
            with patch('sys.argv', argv), patch('yt_dlp.YoutubeDL', return_value=Ydl(data)), contextlib.redirect_stdout(output):
                self.assertEqual(resolve.main(), 0)
            events = [json.loads(line) for line in output.getvalue().splitlines()]
            self.assertEqual(events[-1]['kind'], 'images')
            self.assertEqual(events[-1]['assets'][0]['filename'], 'image-000.jpg')
            self.assertEqual((Path(tmp) / 'image-000.jpg').read_bytes(), data)

    def test_rejects_invalid_images_total_size_and_excess_count(self):
        with tempfile.TemporaryDirectory() as tmp:
            for data, count, limit in [(b'<html>blocked</html>' * 3, 1, 1000),
                                        (b'\xff\xd8\xff' + bytes(90), 2, 100),
                                        (bytes(40), MAX_IMAGES + 1, 10000)]:
                with self.subTest(count=count, limit=limit), self.assertRaises(ValueError):
                    download_images(Transport(data), [{'urls': [Response.url]}] * count, Path(tmp), limit, {}, lambda *a, **k: None)
            self.assertFalse(list(Path(tmp).glob('*.part')))

    def test_rejects_unsafe_urls_and_recognizes_image_formats(self):
        for url in ('file:///etc/passwd', 'http://cdn.example.org/a', 'https://u:p@cdn.example.org/a', 'https://cdn.example.org:444/a'):
            with self.assertRaises(ValueError):
                safe_url(url)
        self.assertEqual(image_ext(b'RIFF0000WEBP'), 'webp')
        self.assertEqual(image_ext(b'\0\0\0\0ftypavif'), 'avif')


if __name__ == '__main__':
    unittest.main()
