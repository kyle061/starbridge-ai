"""Run the bundled imagegen CLI using the selected Starbridge provider only."""

import argparse
import contextlib
import importlib.util
import json
import os
from pathlib import Path
import re
import runpy
import sys
from urllib.parse import urlsplit
from unittest.mock import patch


def runtime_paths(home, windows):
    runtime = home / ".cache/codex-runtimes/codex-primary-runtime/dependencies/python"
    if windows:
        return home / ".cache/starbridge-imagegen/venv/Scripts/python.exe", runtime / "python.exe"
    return home / ".cache/starbridge-imagegen/venv/bin/python", runtime / "bin/python3"


def use_available_runtime():
    dedicated, bundled = runtime_paths(Path.home(), sys.platform == "win32")
    candidates = [dedicated]
    if sys.version_info < (3, 11):
        candidates.append(bundled)
    for candidate in candidates:
        if candidate.is_file() and os.path.normcase(os.path.abspath(candidate)) != os.path.normcase(os.path.abspath(sys.executable)):
            # A venv's executable resolves to its base Python; compare the invoked path.
            os.execv(str(candidate), [str(candidate), *sys.argv])
    if sys.version_info < (3, 11):
        raise ValueError("Use Python 3.11+ or the Codex bundled Python runtime for this launcher.")


def provider_environment(config_path, environment, profile=None):
    import tomllib

    try:
        with config_path.open("rb") as source:
            config = tomllib.load(source)
    except (OSError, ValueError):
        raise ValueError("Cannot read a valid Codex provider config.") from None
    selected = config
    if profile is not None:
        selected = config.get("profiles", {}).get(profile)
        if not isinstance(selected, dict):
            raise ValueError("The requested Codex profile does not exist.")
    name = selected.get("model_provider", config.get("model_provider"))
    provider = config.get("model_providers", {}).get(name, {})
    base_url = provider.get("base_url", "")
    if not isinstance(base_url, str):
        raise ValueError("The selected provider has no valid base_url.")
    try:
        parsed = urlsplit(base_url)
        trusted = (parsed.hostname == "starbridaeai.top" or (parsed.hostname or "").endswith(".starbridaeai.top"))
        valid = parsed.scheme == "https" and trusted and not parsed.username and not parsed.password and not parsed.query and not parsed.fragment and parsed.port in (None, 443)
    except ValueError:
        valid = False
    if not valid:
        raise ValueError("The selected provider is not a Starbridge HTTPS endpoint; no credentials were used.")
    env_key = provider.get("env_key")
    credential = environment.get(env_key, "") if isinstance(env_key, str) else ""
    credential = credential or provider.get("experimental_bearer_token", "")
    if not isinstance(credential, str) or not credential.strip():
        raise ValueError("The selected Starbridge provider has no configured credential.")
    child = dict(environment)
    child["OPENAI_BASE_URL"] = base_url.rstrip("/") + "/"
    child["OPENAI_API_KEY"] = credential.strip()
    child.pop("OPENAI_ORG_ID", None)
    child.pop("OPENAI_PROJECT_ID", None)
    return child, {"provider": name, "base_url": child["OPENAI_BASE_URL"], "credential_configured": True}


def preferred_image_model(model_ids):
    candidates = []
    for model in model_ids:
        if not isinstance(model, str):
            continue
        version = re.fullmatch(r"gpt-image-(\d+(?:\.\d+)*)(?:-[A-Za-z0-9][A-Za-z0-9._-]*)?", model)
        if version is None:
            continue
        parts = tuple(int(part) for part in version[1].split("."))
        while len(parts) > 1 and parts[-1] == 0:
            parts = parts[:-1]
        candidates.append((parts, model))
    if not candidates:
        raise ValueError("This Starbridge key has no available GPT image model; check its group and account model permissions.")
    newest = max(version for version, _ in candidates)
    return min(model for version, model in candidates if version == newest)


def has_explicit_model(args):
    return any(arg == "--model" or arg.startswith("--model=") for arg in args)


def run_bundled_cli(cli, cli_args, environment):
    import openai

    sync_client, async_client = openai.OpenAI, openai.AsyncOpenAI

    def client(*args, **kwargs):
        kwargs.setdefault("max_retries", 0)
        kwargs.setdefault("timeout", 300.0)
        return sync_client(*args, **kwargs)

    def async_image_client(*args, **kwargs):
        kwargs.setdefault("max_retries", 0)
        kwargs.setdefault("timeout", 300.0)
        return async_client(*args, **kwargs)

    args = list(cli_args)
    if args[0] == "generate-batch" and not any(arg == "--max-attempts" or arg.startswith("--max-attempts=") for arg in args):
        args.extend(["--max-attempts", "1"])
    if not has_explicit_model(args) and "--dry-run" not in args:
        with patch.dict(os.environ, environment, clear=True):
            models = sync_client(max_retries=0, timeout=15.0).models.list()
        args.extend(["--model", preferred_image_model(model.id for model in models.data)])
    # These overrides live only in the launcher process; the bundled source is untouched.
    with contextlib.ExitStack() as stack:
        stack.enter_context(patch.dict(os.environ, environment, clear=True))
        stack.enter_context(patch.object(openai, "OpenAI", client))
        stack.enter_context(patch.object(openai, "AsyncOpenAI", async_image_client))
        stack.enter_context(patch.object(sys, "argv", [str(cli), *args]))
        runpy.run_path(str(cli), run_name="__main__")


def main():
    codex_dir = Path(os.environ.get("CODEX_HOME") or Path.home() / ".codex")
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", type=Path, default=codex_dir / "config.toml")
    parser.add_argument("--profile")
    parser.add_argument("--check", action="store_true")
    options, cli_args = parser.parse_known_args()
    if cli_args[:1] == ["--"]:
        cli_args = cli_args[1:]
    try:
        use_available_runtime()
        environment, status = provider_environment(options.config, os.environ, options.profile)
        cli = codex_dir / "skills/.system/imagegen/scripts/image_gen.py"
        status["bundled_cli_available"] = cli.is_file()
        status["sdk_available"] = importlib.util.find_spec("openai") is not None
        status["pillow_available"] = importlib.util.find_spec("PIL") is not None
        status["ready"] = status["bundled_cli_available"] and status["sdk_available"] and status["pillow_available"]
        if options.check:
            print(json.dumps(status))
            return 0 if status["ready"] else 2
        if not status["ready"]:
            raise ValueError("The bundled imagegen CLI, openai SDK or pillow is missing; prepare the dedicated runtime, not a new API key.")
        if not cli_args or cli_args[0] not in ("generate", "edit", "generate-batch"):
            raise ValueError("Pass -- followed by a bundled imagegen command: generate, edit or generate-batch.")
        run_bundled_cli(cli, cli_args, environment)
    except ValueError as error:
        print(str(error), file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
