"""One process per job; JSON events on stdout, no credentials in client responses."""
import argparse
import ipaddress
import json
import os
from pathlib import Path
import socket
import sys
import time
import urllib.parse
import urllib.request


def emit(kind, **data):
    print(json.dumps({"type": kind, **data}, ensure_ascii=False), flush=True)


# Validate the actual addresses returned to connect(), including redirects and CDN requests.
_getaddrinfo = socket.getaddrinfo


def public_addresses(host, port, *args, **kwargs):
    addresses = _getaddrinfo(host, port, *args, **kwargs)
    if not addresses:
        raise OSError("No public address")
    for address in addresses:
        ip = ipaddress.ip_address(address[4][0].split("%")[0])
        mapped = getattr(ip, "ipv4_mapped", None)
        if not (mapped or ip).is_global:
            raise OSError("Private network access blocked")
    return addresses


socket.getaddrinfo = public_addresses


class SilentLogger:
    def debug(self, message):
        pass

    def warning(self, message):
        pass

    def error(self, message):
        pass


def canonical_douyin(url):
    for attempt in range(2):
        parsed = urllib.parse.urlsplit(url)
        host = parsed.hostname or ""
        if not any(host == domain or host.endswith("." + domain) for domain in ("douyin.com", "iesdouyin.com")):
            raise ValueError("Unsupported short link destination")
        parts = parsed.path.strip("/").split("/")
        if parts and parts[0] == "note":
            raise ValueError("Image posts are not supported")
        video_id = urllib.parse.parse_qs(parsed.query).get("modal_id", [None])[0]
        if "video" in parts:
            index = parts.index("video")
            if index + 1 < len(parts):
                video_id = parts[index + 1]
        if video_id and video_id.isdigit():
            return "https://www.douyin.com/video/" + video_id
        if attempt == 0 and host in ("v.douyin.com", "iesdouyin.com", "www.iesdouyin.com"):
            request = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
            with urllib.request.urlopen(request, timeout=20) as response:
                url = response.geturl()
        else:
            break
    return url


def classify_error(message):
    message = message.lower()
    if "image posts are not supported" in message:
        return "NOT_VIDEO", "该链接指向抖音图文作品，当前仅支持视频，请换一个视频链接"
    if any(term in message for term in ["size limit", "duration limit", "file size", "max_filesize"]):
        return "LIMIT", "视频时长或文件大小超出限制，请选择较短的视频或相册兼容模式"
    if any(term in message for term in ["cookie", "sign in", "login", "log in", "authentication"]):
        return "PLATFORM_AUTH", "平台要求有效的访问凭证，当前链接暂时无法解析"
    if "drm" in message:
        return "PROTECTED", "该视频受保护，无法提供下载"
    if any(term in message for term in ["private video", "not available", "unavailable", "removed", "geo"]):
        return "UNAVAILABLE", "视频不可用、已删除或在服务器所在地区无法访问"
    if "format" in message:
        return "NO_FORMAT", "没有找到所选模式的视频源，请尝试另一种保存模式"
    if any(term in message for term in ["timed out", "network", "resolve", "connection"]):
        return "NETWORK", "暂时无法连接视频平台，请稍后重试"
    if "ffmpeg" in message:
        return "DEPENDENCY_MISSING", "服务器缺少 FFmpeg，请联系管理员"
    return "RESOLVE_FAILED", "平台暂未提供可下载的视频源，请更换链接或稍后重试"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--url", required=True)
    parser.add_argument("--platform", required=True)
    parser.add_argument("--mode", choices=["original", "compatible"], required=True)
    parser.add_argument("--directory", required=True)
    parser.add_argument("--max-mb", type=int, required=True)
    parser.add_argument("--max-duration", type=int, required=True)
    parser.add_argument("--cookies")
    args = parser.parse_args()
    try:
        import yt_dlp
        from platform_extractors import install
        from media_compat import ALBUM_MAX_BYTES, album_copy, format_selector, probe
        from yt_dlp.downloader.external import FFmpegFD
        from yt_dlp.utils import DownloadError
    except ImportError:
        emit("error", code="DEPENDENCY_MISSING", message="服务器尚未安装 yt-dlp")
        return 1
    directory = Path(args.directory).resolve()
    def reject_external_download(*unused_args, **unused_kwargs):
        raise DownloadError('External network download protocol is not supported')
    FFmpegFD.real_download = reject_external_download
    max_bytes = args.max_mb * 1024 * 1024
    last_emit = [0.0]

    def progress(event):
        downloaded = event.get("downloaded_bytes") or 0
        if downloaded > max_bytes:
            raise ValueError("file size limit")
        if time.monotonic() - last_emit[0] > 0.5:
            total = event.get("total_bytes") or event.get("total_bytes_estimate") or max_bytes
            emit("progress", value=min(95, downloaded / max(total, 1) * 95))
            last_emit[0] = time.monotonic()

    def limit(info, *, incomplete=False):
        if info.get("is_live") or info.get("live_status") in ("is_live", "is_upcoming"):
            return "Live videos are not supported"
        if (info.get("duration") or 0) > args.max_duration:
            return "Video duration limit exceeded"
        if (info.get("filesize") or 0) > max_bytes:
            return "Video file size limit exceeded"
        return None

    extractors = {
        "douyin": ["Douyin"], "tiktok": ["TikTok.*"],
        "youtube": ["youtube.*", "Youtube.*"], "bilibili": ["Bili.*"],
        "instagram": ["Instagram.*"], "twitter": ["Twitter.*"],
        "vimeo": ["Vimeo.*"], "facebook": ["Facebook.*"], "direct": ["Generic"],
    }
    options = {
        "quiet": True, "no_warnings": True, "logger": SilentLogger(),
        "noplaylist": True, "playlistend": 1,
        "outtmpl": str(directory / "video.%(ext)s"),
        "format": format_selector(args.mode, max_bytes, args.platform == "direct"),
        "merge_output_format": "mp4" if args.mode == "compatible" else "mkv/mp4",
        "max_filesize": max_bytes,
        "socket_timeout": 25, "retries": 2, "fragment_retries": 2,
        "concurrent_fragment_downloads": 2, "hls_prefer_native": True,
        "progress_hooks": [progress], "proxy": "", "cachedir": False,
        "allowed_extractors": extractors.get(args.platform, []),
        "enable_file_urls": False, "usenetrc": False,
        "remote_components": set(),
        "js_runtimes": {"node": {}},
    }
    if args.cookies:
        # Do not let concurrent workers write back to the shared cookie file.
        import shutil
        cookie_copy = directory / "cookies.txt"
        shutil.copyfile(args.cookies, cookie_copy)
        os.chmod(cookie_copy, 0o600)
        options["cookiefile"] = str(cookie_copy)
    try:
        url = canonical_douyin(args.url) if args.platform == "douyin" else args.url
        with yt_dlp.YoutubeDL(options) as ydl:
            install(ydl, args.platform)
            info = ydl.extract_info(url, download=False)
            if not info or info.get("_type") in ("playlist", "multi_video"):
                raise ValueError("Only a single video is supported")
            rejection = limit(info)
            if rejection:
                emit("error", code="LIMIT", message="视频过长、过大或为直播，请选择较短的视频")
                return 1
            emit("metadata", title=info.get("title", "视频"), duration=info.get("duration"))
            emit('phase', value='downloading')
            # Process the already selected formats instead of making a second page request.
            info = ydl.process_ie_result(info, download=True)
            candidates = [p for p in directory.glob("video.*") if p.suffix.lower() in (".mp4", ".webm", ".mkv", ".mov")]
            if len(candidates) != 1:
                raise ValueError("No final video file")
            result = candidates[0]
            if result.stat().st_size > max_bytes:
                emit("error", code="FILE_TOO_LARGE", message="视频超出大小限制")
                return 1
            media = probe(result)
            if media['duration'] > args.max_duration:
                emit("error", code="LIMIT", message="视频时长超出限制")
                return 1
            transcoded = False
            if args.mode == 'compatible' and (not media['compatible'] or result.stat().st_size > min(max_bytes, ALBUM_MAX_BYTES)):
                emit('phase', value='transcoding')
                emit('progress', value=96)
                converted, media = album_copy(result, max_bytes, media['duration'])
                result.unlink()
                result = converted.rename(directory / 'video.mp4')
                transcoded = True
            emit("metadata", title=info.get("title", "视频"), duration=media['duration'])
            emit("complete", filename=result.name, ext=result.suffix[1:], compatible=media['compatible'], transcoded=transcoded)
            return 0
    except Exception as error:
        code, message = classify_error(str(error))
        emit("error", code=code, message=message)
        return 1
    finally:
        if args.cookies:
            (directory / "cookies.txt").unlink(missing_ok=True)


if __name__ == "__main__":
    sys.exit(main())
