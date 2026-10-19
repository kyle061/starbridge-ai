"""Small adapters over yt-dlp's parsers; sessions stay inside a single worker."""
import hashlib
import json
import time
import urllib.parse
import uuid

from yt_dlp.extractor.tiktok import DouyinIE as BaseDouyinIE, TikTokIE as BaseTikTokIE
from yt_dlp.utils import ExtractorError
from yt_dlp.utils.traversal import traverse_obj


class DouyinIE(BaseDouyinIE):
    # Adapted from yt-dlp/yt-dlp#17826, revision edf5ca18f756b1a21f56c87aaf1aef7ef8d3520e.
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
            raise ExtractorError('Fresh platform session cookies are required', expected=True)
        if detail.get('images') and not detail.get('video'):
            raise ExtractorError('Image posts are not supported', expected=True)
        return self._parse_aweme_video_app(detail)


class TikTokIE(BaseTikTokIE):
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
            detail = self._get_universal_data(webpage, video_id).get('webapp.video-detail') or {}
            if detail:
                return traverse_obj(detail, ('itemInfo', 'itemStruct', {dict})), detail.get('statusCode', -1)
        return super()._extract_web_data_and_status(url, video_id, fatal=fatal)


def install(ydl, platform):
    if platform == 'douyin':
        ydl.add_info_extractor(DouyinIE())
    elif platform == 'tiktok':
        ydl.add_info_extractor(TikTokIE())
