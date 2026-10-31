"""Small adapters over yt-dlp's parsers; sessions stay inside a single worker."""
import hashlib
import json
import time
import urllib.parse
import uuid

from yt_dlp.extractor.tiktok import DouyinIE as BaseDouyinIE, TikTokIE as BaseTikTokIE
from yt_dlp.extractor.instagram import InstagramIE as BaseInstagramIE
from yt_dlp.extractor.twitter import TwitterIE as BaseTwitterIE
from yt_dlp.utils import ExtractorError
from yt_dlp.utils.traversal import traverse_obj


class DouyinIE(BaseDouyinIE):
    # Adapted from yt-dlp/yt-dlp#17826, revision edf5ca18f756b1a21f56c87aaf1aef7ef8d3520e.
    _VALID_URL = r'https?://(?:www\.)?douyin\.com/(?:video|note)/(?P<id>\d+)'
    _SEC_SALT = 'A96D855A08C0A9707F8BEF0D9A527E4E'

    def _prepare_session(self):
        if not self._get_cookies(self._WEBPAGE_HOST).get('passport_csrf_token'):
            self._set_cookie('.douyin.com', 'passport_csrf_token', uuid.uuid4().hex)
        if not self._get_cookies(self._WEBPAGE_HOST).get('ttwid'):
            self._request_webpage(self._WEBPAGE_HOST + 'ttwid/register', None,
                                  data=json.dumps({'aid': 6383, 'service': 'www.douyin.com'}).encode(),
                                  headers={'content-type': 'application/json'})
        if not self._get_cookies(self._WEBPAGE_HOST).get('ttwid'):
            raise ExtractorError('Unable to establish anonymous platform session', expected=True)

    def _signed_query(self, query):
        names = ('uifid', 'uifid_temp', 'uifidtemp', 'UIFID', 'UIFID_TEMP', 'UIFIDTEMP')
        uifid = traverse_obj(self._get_cookies(self._WEBPAGE_HOST), (names, 'value'), get_all=False)
        if not uifid:
            response = self._request_webpage(self._WEBPAGE_HOST, None)
            uifid = response.headers.get('x-tt-trace-host')
        if not uifid:
            raise ExtractorError('Fresh platform session cookies are required', expected=True)
        timestamp = str(int(time.time()))
        query = {**query, 'uifid': uifid, 'timestamp': timestamp}
        encoded = urllib.parse.urlencode(query)
        signature = hashlib.md5(f'{uifid}_{timestamp}_{self._SEC_SALT}_{encoded}'.encode()).hexdigest()
        return {**query, 'x-secsdk-web-signature': signature}

    def _real_extract(self, url):
        video_id = self._match_id(url)
        self._prepare_session()
        response = self._download_json(self._WEBPAGE_HOST + 'aweme/v1/web/aweme/detail/', video_id,
                                       query=self._signed_query({'aweme_id': video_id}))
        detail = traverse_obj(response, ('aweme_detail', {dict}))
        if not detail:
            if response.get('filter_detail') or response.get('status_code') == 0:
                raise ExtractorError('Post unavailable or removed', expected=True)
            raise ExtractorError('Fresh platform session cookies are required', expected=True)
        if detail.get('images'):
            return image_post(video_id, detail.get('desc'), detail['images'], url)
        return self._parse_aweme_video_app(detail)


def image_post(post_id, title, images, referer):
    sources = []
    for image in images:
        # Preserve the platform's full-size source, never substitute a thumbnail.
        source = image.get('display_image') or image.get('imageURL') or image
        urls = source.get('url_list') or source.get('download_url_list') or source.get('urlList') or []
        urls = [u for u in urls if isinstance(u, str) and u.startswith('https://')]
        if not urls:
            raise ExtractorError('Image source unavailable', expected=True)
        sources.append({'urls': urls})
    return {'id': post_id, 'title': title or '图文作品', '_type': 'playlist',
            'entries': [], '_liuying_images': sources, 'http_headers': {'Referer': referer}}


class TikTokIE(BaseTikTokIE):
    _VALID_URL = r'https?://(?:www\.)?tiktokv?\.com/(?:embed|(?:share|@(?P<user_id>[\w.-]+)?)/(?:video|photo))/(?P<id>\d+)'

    def _real_extract(self, url):
        if '/photo/' in url:
            post_id = self._match_id(url)
            data, status = self._extract_web_data_and_status(url, post_id)
            if data and status == 0:
                return self._parse_aweme_video_web(data, url, post_id)
            raise ExtractorError(f'Post unavailable, status code {status}', expected=True)
        return super()._real_extract(url)

    def _parse_aweme_video_web(self, data, url, video_id):
        images = (data.get('imagePost') or {}).get('images') or []
        if images:
            return image_post(video_id, data.get('desc'), images, url)
        return super()._parse_aweme_video_web(data, url, video_id)

    @staticmethod
    def _create_url(user_id, video_id):
        if not user_id or user_id == '_':
            return f'https://www.tiktok.com/share/video/{video_id}/'
        return BaseTikTokIE._create_url(user_id, video_id)

    def _extract_web_data_and_status(self, url, video_id, fatal=True):
        result = self._download_webpage_handle(url, video_id, fatal=fatal, headers={
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36',
            'Accept-Language': 'en-US,en;q=0.9',
        }, impersonate=False)
        if result:
            webpage, response = result
            if urllib.parse.urlsplit(response.url).path == '/login':
                self.raise_login_required('TikTok requires an authorized platform login')
            detail = self._get_universal_data(webpage, video_id).get('webapp.video-detail') or self._get_universal_data(webpage, video_id).get('webapp.photo-detail') or {}
            if detail:
                return traverse_obj(detail, ('itemInfo', 'itemStruct', {dict})), detail.get('statusCode', -1)
        return super()._extract_web_data_and_status(url, video_id, fatal=fatal)


class InstagramIE(BaseInstagramIE):
    def _extract_product(self, product_info, video_id=None, get_comments=True):
        product = product_info[0] if isinstance(product_info, list) else product_info
        media = product.get('carousel_media') or [product]
        if media and all(not item.get('video_versions') and not item.get('video_dash_manifest') for item in media):
            images = []
            for item in media:
                candidates = (item.get('image_versions2') or {}).get('candidates') or []
                candidates = sorted(candidates, key=lambda c: (c.get('width') or 0) * (c.get('height') or 0), reverse=True)
                images.append({'url_list': [c['url'] for c in candidates if c.get('url')]})
            return image_post(str(video_id or product.get('pk')), (product.get('caption') or {}).get('text'), images,
                              'https://www.instagram.com/')
        return super()._extract_product(product_info, video_id, get_comments)


class TwitterIE(BaseTwitterIE):
    def _real_extract(self, url):
        post_id = self._match_id(url)
        status = self._extract_status(post_id)
        media = (status.get('extended_entities') or {}).get('media') or []
        if media and all(item.get('type') == 'photo' for item in media):
            images = []
            for item in media:
                source = item.get('media_url_https')
                if not source:
                    raise ExtractorError('Image source unavailable', expected=True)
                parsed = urllib.parse.urlsplit(source)
                query = dict(urllib.parse.parse_qsl(parsed.query))
                query['name'] = 'orig'
                images.append({'url_list': [urllib.parse.urlunsplit(parsed._replace(query=urllib.parse.urlencode(query)))]})
            return image_post(post_id, status.get('full_text') or status.get('text'), images, 'https://x.com/')
        # Reuse the fetched status; avoid a second API call for video posts.
        self._extract_status = lambda unused: status
        return super()._real_extract(url)


def install(ydl, platform):
    if platform == 'douyin':
        ydl.add_info_extractor(DouyinIE())
    elif platform == 'tiktok':
        ydl.add_info_extractor(TikTokIE())
    elif platform == 'instagram':
        ydl.add_info_extractor(InstagramIE())
    elif platform == 'twitter':
        ydl.add_info_extractor(TwitterIE())
