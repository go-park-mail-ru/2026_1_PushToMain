#!/usr/bin/env python3
"""
Генерирует файл targets для vegeta из сохранённых сессий.

Использование:
    python3 make_targets.py --host http://localhost:8082 --out targets_inbox.txt
"""

import argparse
import json
import random

DEFAULT_HOST    = "http://localhost:8082"
DEFAULT_COUNT   = 1000
DEFAULT_OUT     = "targets_inbox.txt"


def load_sessions(path: str) -> list[dict]:
    with open(path) as f:
        return json.load(f)


def get_cookie_str(session: dict) -> str:
    """Извлекает session_token из cookies."""
    cookies = session.get("cookies", {})
    if "session_token" in cookies:
        return f"session_token={cookies['session_token']}"
    # fallback для старого формата
    if "session" in session:
        return f"session_id={session['session']}"
    return ""


def make_targets_inbox(host: str, sessions: list[dict], n: int) -> list[str]:
    lines = []
    for _ in range(n):
        s = random.choice(sessions)
        cookie = get_cookie_str(s)
        if not cookie:
            continue
        lines += [
            f"GET {host}/api/v1/email/inbox",
            f"Cookie: {cookie}",
            "",
        ]
    return lines


def make_targets_by_id(host: str, sessions: list[dict], n: int,
                       max_id: int = 100_000) -> list[str]:
    lines = []
    for _ in range(n):
        s = random.choice(sessions)
        eid = random.randint(1, max_id)
        cookie = get_cookie_str(s)
        if not cookie:
            continue
        lines += [
            f"GET {host}/api/v1/email/emails/{eid}",
            f"Cookie: {cookie}",
            "",
        ]
    return lines


def main():
    parser = argparse.ArgumentParser(description="Генерация targets для vegeta")
    parser.add_argument("--host", default=DEFAULT_HOST)
    parser.add_argument("--sessions", default="pref_test/test_sessions.json")
    parser.add_argument("--count", default=DEFAULT_COUNT, type=int)
    parser.add_argument("--out", default=DEFAULT_OUT)
    parser.add_argument("--mode", default="inbox",
                        choices=["inbox", "by_id"])
    parser.add_argument("--max-id", default=100_000, type=int)
    args = parser.parse_args()

    sessions = load_sessions(args.sessions)
    if not sessions:
        raise SystemExit("Нет сессий в test_sessions.json")

    if args.mode == "inbox":
        lines = make_targets_inbox(args.host, sessions, args.count)
    else:
        lines = make_targets_by_id(args.host, sessions, args.count, args.max_id)

    with open(args.out, "w") as f:
        f.write("\n".join(lines) + "\n")

    print(f"Записано {args.count} targets → {args.out}")


if __name__ == "__main__":
    main()