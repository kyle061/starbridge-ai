# Platform Adapter Notes

The project pins yt-dlp in `requirements.txt` and reuses its metadata parsers, format selection, network stack and downloaders. `platform_extractors.py` registers small subclasses with the same extractor keys. The public-address DNS guard remains active; this project does not add a native cURL transport that would avoid that guard.

## Douyin

The anonymous session setup and request signature are adapted from [yt-dlp pull request 17826](https://github.com/yt-dlp/yt-dlp/pull/17826), revision `edf5ca18f756b1a21f56c87aaf1aef7ef8d3520e`. This proposed upstream change was not yet merged when integrated on 2026-10-18. yt-dlp is distributed under the Unlicense; see its [license](https://github.com/yt-dlp/yt-dlp/blob/master/LICENSE).

The adapter obtains the platform's normal anonymous session cookies and request identifier, then requests video metadata with the current signature. It does not use stored account credentials unless the operator explicitly configures a private cookie file. The fixed signature salt is a public protocol constant, not a deployment credential. Private videos, authentication requirements and platform restrictions remain applicable.

Remove the local override after verifying an upstream release contains an equivalent working implementation. Keep actual platform links in private acceptance records rather than embedding user identifiers or cookies in tests.

## TikTok

Share URLs without a creator name retain the official `/share/video/<id>/` path rather than becoming `/@_/video/<id>`. The adapter first requests the ordinary public desktop page and passes its hydration data to yt-dlp's existing parser. A mobile page containing only app-launch instructions is not treated as video metadata. When the detail data is absent, the existing upstream extraction path handles the failure or access requirement.

## Album Copies

H.264 may be labeled `h264` or `avc1` by different upstream extractors. Both labels are accepted. Existing compatible MP4 files within the size cap are copied without transcoding. When needed, `media_compat.py` creates a separate H.264/AAC MP4 below the album size cap, verifies codecs and duration, and leaves original mode unmodified. Worker cancellation terminates the whole process group, including FFmpeg.

## Image posts (2026-10-31)

The adapters return a private image manifest before yt-dlp video format processing. Douyin note URLs use the same post detail API; TikTok `/photo/` and video-path slideshow metadata read `imagePost.images`. Instagram pure image products use the largest `image_versions2` candidates. X pure photo posts request `name=orig` from the declared photo URL. Covers and thumbnails are never silently used as post images. Mixed video/image carousels are not yet supported.

`images.py` downloads through the same cookie-aware yt-dlp transport and public-address DNS guard. It validates HTTPS source URLs, image signatures, image count, individual and total size caps, and stores bytes without re-encoding. ZIP archives contain these exact downloaded bytes. Each asset URL signs the job, resource key and expiration together. `album.py` creates requested album copies in the bounded worker queue, retaining ready originals when conversion fails.

The old `mode` API is retained for earlier uploaded clients. Current clients omit it and the server defaults to `original`. Album preparation is a separate authenticated operation and never replaces the original download URLs.
