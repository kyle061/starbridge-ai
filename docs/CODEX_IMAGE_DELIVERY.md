# Codex generated-image delivery

Some Codex clients retain the Responses `image_generation_call.result` base64 in
their local transcript without creating a visible image attachment. Starbridge
keeps that original result and adds an ordinary assistant message containing a
Markdown preview and a download link. This runs only for detected Codex clients
on Responses HTTP/SSE and WebSocket paths, including HTTP bridge and WS passthrough.
It does not generate another image or change upstream usage accounting.

Completed PNG, JPEG and WebP results are stored in `generated-images` beneath the
deployment's existing data directory (`DATA_DIR`). Each URL has a random 256-bit
bearer token; anyone holding that link can retrieve that one image for 24 hours.
Links contain no API key or account ID. Save files locally before expiry. Download
responses disable caching and referrer transmission. Never publish those URLs in
public logs or messages if the image is private.

The local store accepts at most 32 MiB per image, 16 images per response, 4096
files and 1 GiB of unexpired images. Expired files are removed on the next image
save. When storage is unavailable/full, the original API image remains intact and
the assistant message explicitly reports the missing download file. It does not
request another generation. Multiple gateway replicas must share this data
directory or route image downloads to the same replica. The public reverse proxy
must forward `/images/generated/*` to the gateway.

`server.frontend_url` is used as the public URL when configured; otherwise the
request host and HTTPS forwarding information are used. A client that restricts
remote image previews can still open the download link and render the downloaded
file locally. This compatibility output does not modify the Codex application.

To replay a previously recovered image without calling a model:

```sh
cd backend
CODEX_IMAGE_REPLAY_FILE=/absolute/path/to/image.png go test ./internal/handler -run '^TestCodexImageDeliveryRealImageReplay$' -count=1
```

The replay checks both buffered JSON and chunked SSE, preserves the original
base64 and usage, downloads the generated link, compares every byte, and decodes
the downloaded PNG. The image itself is not added to the repository.
