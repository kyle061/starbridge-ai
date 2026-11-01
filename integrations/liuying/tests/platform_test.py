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
from album import prepare

try:
    import yt_dlp
    from platform_extractors import TikTokIE, BaseTikTokIE, DouyinIE, InstagramIE, TwitterIE, image_post
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

    def test_gallery_uses_full_size_sources_and_accepts_photo_urls(self):
        self.assertTrue(TikTokIE.suitable('https://www.tiktok.com/@creator/photo/123456'))
        self.assertTrue(DouyinIE.suitable('https://www.douyin.com/note/123456'))
        images = [{'imageURL': {'urlList': ['https://cdn.example.com/original.webp']},
                   'thumbnail': {'urlList': ['https://cdn.example.com/thumb.webp']}}]
        with yt_dlp.YoutubeDL({'quiet': True}) as ydl:
            result = TikTokIE(ydl)._parse_aweme_video_web({'desc': 'photos', 'imagePost': {'images': images}},
                                                         'https://www.tiktok.com/@creator/photo/123456', '123456')
        self.assertEqual(result['_liuying_images'][0]['urls'], ['https://cdn.example.com/original.webp'])
        with self.assertRaises(Exception):
            image_post('123', 'missing', [{'url_list': []}], 'https://www.douyin.com/')

    def test_douyin_web_client_identity_avoids_images_base_filter(self):
        def detail_response(url, post_id, query):
            if query.get('aid') != 6383 or query.get('device_platform') != 'webapp' or not query.get('version_code'):
                return {'status_code': 0, 'filter_detail': {'filter_reason': 'images_base'}}
            return {'status_code': 0, 'aweme_detail': {'aweme_id': post_id, 'desc': 'photos', 'images': [
                {'url_list': ['https://cdn.example.com/full.webp'],
                 'thumbnail': {'url_list': ['https://cdn.example.com/thumb.webp']}}]}}

        with yt_dlp.YoutubeDL({'quiet': True}) as ydl:
            extractor = DouyinIE(ydl)
            with patch.object(extractor, '_prepare_session'), patch.object(extractor, '_signed_query', side_effect=lambda query: query), \
                    patch.object(extractor, '_download_json', side_effect=detail_response):
                result = extractor._real_extract('https://www.douyin.com/note/123456')
            self.assertEqual(result['_liuying_images'][0]['urls'], ['https://cdn.example.com/full.webp'])

            with patch.object(extractor, '_prepare_session'), patch.object(extractor, '_signed_query', side_effect=lambda query: query), \
                    patch.object(extractor, '_download_json', return_value={'status_code': 0, 'filter_detail': {'filter_reason': 'private'}}), \
                    patch.object(extractor, '_parse_aweme_video_app') as video_parser:
                with self.assertRaisesRegex(Exception, 'unavailable'):
                    extractor._real_extract('https://www.douyin.com/note/123456')
                video_parser.assert_not_called()

    def test_douyin_web_identity_keeps_video_parsing(self):
        detail = {'aweme_id': '123456', 'video': {'play_addr': {'url_list': ['https://cdn.example.com/video.mp4']}}}
        with yt_dlp.YoutubeDL({'quiet': True}) as ydl:
            extractor = DouyinIE(ydl)
            with patch.object(extractor, '_prepare_session'), patch.object(extractor, '_signed_query', side_effect=lambda query: query), \
                    patch.object(extractor, '_download_json', return_value={'status_code': 0, 'aweme_detail': detail}), \
                    patch.object(extractor, '_parse_aweme_video_app', return_value={'id': '123456'}) as video_parser:
                self.assertEqual(extractor._real_extract('https://www.douyin.com/video/123456'), {'id': '123456'})
                video_parser.assert_called_once_with(detail)

    def test_instagram_and_x_keep_full_size_image_sources(self):
        with yt_dlp.YoutubeDL({'quiet': True}) as ydl:
            instagram = InstagramIE(ydl)._extract_product({'pk': '123', 'image_versions2': {'candidates': [
                {'url': 'https://cdn.example.com/small.jpg', 'width': 100, 'height': 100},
                {'url': 'https://cdn.example.com/large.jpg', 'width': 2000, 'height': 2000}]}}, get_comments=False)
            self.assertEqual(instagram['_liuying_images'][0]['urls'][0], 'https://cdn.example.com/large.jpg')
            twitter = TwitterIE(ydl)
            with patch.object(twitter, '_extract_status', return_value={'extended_entities': {'media': [
                {'type': 'photo', 'media_url_https': 'https://pbs.twimg.com/media/test.jpg?name=small'}]}}):
                result = twitter._real_extract('https://x.com/test/status/123')
                self.assertEqual(result['_liuying_images'][0]['urls'][0], 'https://pbs.twimg.com/media/test.jpg?name=orig')


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

    @unittest.skipUnless(shutil.which('ffmpeg') and shutil.which('ffprobe'), 'FFmpeg and FFprobe required')
    def test_webp_album_copy_keeps_original_image_unchanged(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / 'image-000.webp'
            subprocess.run(['ffmpeg', '-v', 'error', '-f', 'lavfi', '-i', 'color=c=red:size=64x64',
                            '-frames:v', '1', '-threads', '1', '-y', str(source)], check=True, capture_output=True)
            original_hash = hashlib.sha256(source.read_bytes()).digest()
            assets = prepare([{'file': str(source), 'kind': 'image', 'compatible': False}], 1048576)
            result = Path(directory) / assets[0]['filename']
            self.assertTrue(result.read_bytes().startswith(b'\xff\xd8\xff'))
            self.assertEqual(hashlib.sha256(source.read_bytes()).digest(), original_hash)


if __name__ == '__main__':
    unittest.main()
