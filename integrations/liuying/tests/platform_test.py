import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parents[1] / 'worker'))
from media_compat import album_copy, format_selector, probe

try:
    import yt_dlp
    from platform_extractors import TikTokIE, BaseTikTokIE
except ImportError:
    yt_dlp = None


@unittest.skipIf(yt_dlp is None, 'Install worker requirements to test platform adapters')
class PlatformTests(unittest.TestCase):
    def selected_format(self, formats):
        with yt_dlp.YoutubeDL({'quiet': True, 'check_formats': False,
                              'format': format_selector('compatible', 1024 * 1024 * 1024)}) as ydl:
            return ydl.process_ie_result({'id': 'test', 'title': 'test', 'formats': formats}, download=False)

    def test_selects_h264_or_avc1_without_requiring_transcoding(self):
        for codec in ('h264', 'avc1.64001f'):
            with self.subTest(codec=codec):
                result = self.selected_format([
                    {'format_id': 'compatible', 'url': 'https://media.example.com/a.mp4', 'ext': 'mp4', 'vcodec': codec, 'acodec': 'aac', 'height': 720, 'filesize': 3000000},
                    {'format_id': 'hevc', 'url': 'https://media.example.com/b.mp4', 'ext': 'mp4', 'vcodec': 'h265', 'acodec': 'aac', 'height': 1080, 'filesize': 4000000},
                ])
                self.assertEqual(result['format_id'], 'compatible')

    def test_oversized_h264_falls_back_to_a_source_for_conversion(self):
        result = self.selected_format([
            {'format_id': 'oversized', 'url': 'https://media.example.com/a.mp4', 'ext': 'mp4', 'vcodec': 'h264', 'acodec': 'aac', 'height': 720, 'filesize': 600000000},
            {'format_id': 'hevc', 'url': 'https://media.example.com/b.mp4', 'ext': 'mp4', 'vcodec': 'h265', 'acodec': 'aac', 'height': 1080, 'filesize': 200000000},
        ])
        self.assertEqual(result['format_id'], 'hevc')

    def test_tiktok_share_urls_keep_a_valid_share_path(self):
        self.assertEqual(TikTokIE._create_url(None, '123'), 'https://www.tiktok.com/share/video/123/')
        self.assertEqual(TikTokIE._create_url('_', '123'), 'https://www.tiktok.com/share/video/123/')
        self.assertEqual(TikTokIE._create_url('creator', '123'), 'https://www.tiktok.com/@creator/video/123')

    def test_reads_public_tiktok_detail_but_does_not_accept_mobile_scaffold(self):
        detail = {'statusCode': 0, 'itemInfo': {'itemStruct': {'id': '123', 'video': {'playAddr': 'https://media.example.com/a.mp4'}}}}
        data = {'__DEFAULT_SCOPE__': {'webapp.video-detail': detail}}
        with yt_dlp.YoutubeDL({'quiet': True}) as ydl:
            extractor = TikTokIE(ydl)
            page = '<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__">' + json.dumps(data) + '</script>'
            with patch.object(extractor, '_download_webpage_handle', return_value=(page, SimpleNamespace(url='https://www.tiktok.com/share/video/123/'))):
                video, status = extractor._extract_web_data_and_status('https://www.tiktok.com/share/video/123/', '123')
                self.assertEqual(status, 0)
                self.assertEqual(video['id'], '123')
            with patch.object(extractor, '_download_webpage_handle', return_value=('<html></html>', SimpleNamespace(url='https://www.tiktok.com/share/video/123/'))), patch.object(BaseTikTokIE, '_extract_web_data_and_status', return_value=({}, -1)) as fallback:
                self.assertEqual(extractor._extract_web_data_and_status('https://www.tiktok.com/share/video/123/', '123'), ({}, -1))
                fallback.assert_called_once()


class ConversionTests(unittest.TestCase):
    @unittest.skipUnless(shutil.which('ffmpeg') and shutil.which('ffprobe'), 'FFmpeg and FFprobe required')
    def test_creates_bounded_album_copy_without_modifying_original(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / 'source.mkv'
            subprocess.run(['ffmpeg', '-v', 'error', '-f', 'lavfi', '-i', 'testsrc=size=320x180:rate=12',
                            '-f', 'lavfi', '-i', 'sine=frequency=440:sample_rate=44100', '-t', '3',
                            '-c:v', 'mpeg4', '-c:a', 'aac', '-y', str(source)], check=True, capture_output=True)
            original_hash = hashlib.sha256(source.read_bytes()).digest()
            metadata = probe(source)
            self.assertFalse(metadata['compatible'])
            output, converted = album_copy(source, 300000, metadata['duration'])
            self.assertTrue(converted['compatible'])
            self.assertLessEqual(output.stat().st_size, 300000)
            self.assertAlmostEqual(converted['duration'], metadata['duration'], delta=0.2)
            self.assertEqual(hashlib.sha256(source.read_bytes()).digest(), original_hash)


if __name__ == '__main__':
    unittest.main()
