"""Install the Starbridge image skill and its runtime without changing provider config."""

import argparse
import os
from pathlib import Path
import shutil
import subprocess
import sys
import venv


def install(codex_home):
    source = Path(__file__).parent / "starbridge-imagegen"
    destination = codex_home / "skills/starbridge-imagegen"
    runtime = Path.home() / ".cache/starbridge-imagegen/venv"
    python = runtime / ("Scripts/python.exe" if sys.platform == "win32" else "bin/python")
    if not python.is_file():
        venv.EnvBuilder(with_pip=True).create(runtime)
    subprocess.run([str(python), "-m", "pip", "install", "openai", "pillow"], check=True)
    for relative in ("SKILL.md", "scripts/with_current_provider.py"):
        target = destination / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source / relative, target)
    print(f"Installed Starbridge image skill: {destination}", flush=True)
    environment = dict(os.environ, CODEX_HOME=str(codex_home))
    return subprocess.run(
        [str(python), str(destination / "scripts/with_current_provider.py"), "--check"],
        env=environment,
        check=False,
    ).returncode


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--codex-home", type=Path, default=Path(os.environ.get("CODEX_HOME") or Path.home() / ".codex"))
    options = parser.parse_args()
    if sys.version_info < (3, 11):
        print("Use Python 3.11+ to install the image skill.", file=sys.stderr)
        return 2
    try:
        return install(options.codex_home.expanduser().resolve())
    except (OSError, subprocess.CalledProcessError) as error:
        print(f"Image skill installation failed: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
