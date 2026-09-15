#!/usr/bin/env python3
"""Generate first-install secrets without printing them or overwriting an existing .env."""
import argparse
import os
from pathlib import Path
import re
import secrets


def create_env(directory: Path, admin_email: str) -> Path:
    # Restrict interpolation/newline characters: the result is a Docker Compose env file.
    if not re.fullmatch(r"[A-Za-z0-9._+%-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}", admin_email):
        raise ValueError("Provide a valid administrator email address.")
    template = (directory / ".env.example").read_text(encoding="utf-8")
    values = {key: secrets.token_hex(32) for key in (
        "ADMIN_PASSWORD", "POSTGRES_PASSWORD", "REDIS_PASSWORD", "JWT_SECRET", "TOTP_ENCRYPTION_KEY"
    )}
    values["ADMIN_EMAIL"] = admin_email
    lines = []
    for line in template.splitlines():
        key = line.partition("=")[0]
        lines.append(f"{key}={values[key]}" if key in values else line)
    path = directory / ".env"
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, "w", encoding="utf-8") as output:
        output.write("\n".join(lines) + "\n")
    return path


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--admin-email", required=True)
    args = parser.parse_args()
    try:
        path = create_env(Path(__file__).resolve().parent, args.admin_email.strip())
    except (FileExistsError, ValueError) as error:
        parser.exit(1, f"Not changed: {error}\n")
    print(f"Created {path} (owner read/write only).")
    print("Open .env locally to read ADMIN_PASSWORD. Start with: docker compose up -d --build")
