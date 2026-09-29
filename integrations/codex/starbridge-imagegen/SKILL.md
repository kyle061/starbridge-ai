---
name: starbridge-imagegen
description: Generate or edit images through an already configured Starbridge Codex provider when native image tools are unavailable. Reuses the current provider configuration without asking for a new API key or an OpenArt connection. Applies to image requests using Starbridge, not to unrelated providers or text-only tasks.
---

# Starbridge Image Generation

Use this route for an image the user has requested through their existing Starbridge provider. Prefer a callable built-in image tool when available. Missing built-in `image_gen` does not establish that the configured provider lacks image support. Do not automatically introduce Ozon or other domain design skills into a generic image request. Do not narrate the skill, provider, credentials, routing or configuration to the user during normal generation. Deliver the image and any answer requested by the user, without adding platform-specific text to the model's answer.

Run the configuration check first; it makes no API request and never prints a key:

```sh
python3 ~/.codex/skills/starbridge-imagegen/scripts/with_current_provider.py --check
```

This helper reads the currently selected provider from `CODEX_HOME/config.toml`, or `~/.codex/config.toml`. It accepts only the configured Starbridge HTTPS endpoint. It resolves that provider's `env_key` or `experimental_bearer_token`; it does not scan other providers, browser storage or unrelated credentials.

When the user explicitly asks to generate or edit an image while their selected provider is Starbridge, that request authorizes sending the prompt and any source image they supplied for that edit to the configured Starbridge endpoint for this image. Use the available image tool or this provider route without a second conversational question about CLI/API mode or prompt transfer. Do not infer authorization from a configuration check, troubleshooting question, prompt-writing request, unrelated file, or a different provider. Platform-level network approvals still apply: if an approval is required or denied, honor the result and report the blocker without trying to route around it.

The launcher calls the existing bundled imagegen CLI with the provider URL and credential set only in its own process. It leaves the config file, shell profile and bundled CLI unchanged. Read the imagegen skill's prompt/output guidance for the requested image; do not apply its generic missing-key setup steps when this provider check succeeds.

```sh
python3 ~/.codex/skills/starbridge-imagegen/scripts/with_current_provider.py -- generate \
  --prompt-file /absolute/path/prompt.txt \
  --size 1024x1024 --quality medium \
  --out /absolute/workspace/path/output/imagegen/image.png
```

Without `--model`, the launcher makes a non-billable `/v1/models` lookup for the current key and chooses the highest authorized GPT image version (2.5 before 2). Pass `--model` when the user requests a particular model or the group default image model is supplied in the request; this also lets the user select 2 when both 2 and 2.5 are available. Editing uses `-- edit --image /absolute/input.png --prompt-file ... --out ...`. Pass only controls supported by the bundled CLI. Keep output paths in the current workspace and do not overwrite an existing file unless requested. Do not silently substitute a different provider or image model after an upstream rejection.

After the CLI returns, open the saved file with `view_image`, verify the result, and embed its absolute local path in the final reply. A prompt alone is not a generated image. On a failed or timed-out generation, inspect the error and any existing output before retrying; avoid duplicate billable requests.

If the check reports a missing SDK, create the dedicated runtime once using a Python 3.11+ runtime (the Codex bundled Python is suitable):

```sh
python3 -m venv ~/.cache/starbridge-imagegen/venv
~/.cache/starbridge-imagegen/venv/bin/python -m pip install openai pillow
```

Use a suitable Python executable in place of `python3` for runtime creation. The launcher automatically uses this dedicated environment thereafter. Missing actual provider credentials or a group/model permission rejection must be reported precisely; do not misreport them as a requirement for another image plugin.
