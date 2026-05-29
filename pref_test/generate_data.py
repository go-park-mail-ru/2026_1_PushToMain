#!/usr/bin/env python3
import argparse
import random
import string
import json
import time
import urllib.request
import urllib.error
from concurrent.futures import ThreadPoolExecutor, as_completed
from threading import Lock

DEFAULT_HOST = "http://localhost:8081"
DEFAULT_EMAIL_HOST = "http://localhost:8082"

def rand_str(n: int) -> str:
    return ''.join(random.choices(string.ascii_lowercase, k=n))

def http_request(url: str, method: str = "GET", data: dict = None, cookies: dict = None, csrf_token: str = None) -> tuple[int, dict, dict]:
    body = json.dumps(data).encode() if data else None
    req = urllib.request.Request(url, data=body, method=method)
    req.add_header("Content-Type", "application/json")
    
    if cookies:
        cookie_str = "; ".join([f"{k}={v}" for k, v in cookies.items()])
        req.add_header("Cookie", cookie_str)
    
    if csrf_token:
        req.add_header("X-CSRF-Token", csrf_token)
    
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            resp_cookies = {}
            for cookie in resp.headers.get_all("Set-Cookie", []):
                if '=' in cookie:
                    key = cookie.split('=')[0]
                    value = cookie.split('=')[1].split(';')[0]
                    resp_cookies[key] = value
            body_data = json.loads(resp.read()) if resp.getcode() != 204 else {}
            return resp.getcode(), body_data, resp_cookies
    except urllib.error.HTTPError as e:
        print(f"HTTPError: {e.code} {url}")
        return e.code, {}, {}
    except Exception as e:
        print(f"Request error: {e} {url}")
        return 0, {}, {}

def get_csrf_token(host: str, cookies: dict) -> str:
    """Получает CSRF токен."""
    status, body, _ = http_request(f"{host}/api/v1/user/csrf", "GET", cookies=cookies)
    if status == 200:
        return body.get("csrf_token", "")
    return ""

def register_user(host: str) -> tuple[str, dict, str] | None:
    """Регистрирует пользователя, возвращает (email, cookies, csrf_token)."""
    email = f"{rand_str(8)}@e-smail.ru"
    password = "12345678"
    
    # Регистрация
    signup_data = {
        "email": email,
        "password": password,
        "name": rand_str(6).capitalize(),
        "surname": rand_str(8).capitalize()
    }
    status, _, _ = http_request(f"{host}/api/v1/user/signup", "POST", signup_data)
    if status not in (200, 201):
        return None
    
    # Логин
    login_data = {"email": email, "password": password}
    status, _, cookies = http_request(f"{host}/api/v1/user/signin", "POST", login_data)
    if status not in (200, 201):
        return None
    
    # Получаем CSRF токен
    csrf_token = get_csrf_token(host, cookies)
    if not csrf_token:
        print(f"Warning: Could not get CSRF token for {email}")
    
    return email, cookies, csrf_token

def send_email(host: str, cookies: dict, csrf_token: str, receiver: str) -> bool:
    """Отправляет письмо, возвращает True при успехе."""
    payload = {
        "header": "Test Message",
        "body": "This is a test body",
        "receivers": [receiver],
    }
    status, _, _ = http_request(f"{host}/api/v1/email/send", "POST", payload, cookies, csrf_token)
    return status in (200, 201)

def create_users(host: str, n: int) -> list:
    print(f"[1/2] Создаём {n} тестовых пользователей...")
    users = []
    for i in range(n):
        result = register_user(host)
        if result:
            users.append(result)
            print(f"  Created {len(users)}/{n}: {result[0]}")
        time.sleep(0.1)
    return users

def generate_emails(host: str, users: list, total: int, workers: int):
    print(f"[2/2] Генерируем {total} писем ({workers} воркеров)...")
    success = 0
    fail = 0
    
    def worker(_):
        nonlocal success, fail
        # Выбираем случайного отправителя
        sender_email, sender_cookies, csrf_token = random.choice(users)
        # Выбираем случайного получателя (из того же списка)
        receiver_email, _, _ = random.choice(users)  # ← используем существующего пользователя
        if send_email(host, sender_cookies, csrf_token, receiver_email):
            success += 1
        else:
            fail += 1
            print(f"Failed to send from {sender_email} to {receiver_email}")
    
    with ThreadPoolExecutor(max_workers=workers) as pool:
        list(pool.map(worker, range(total)))
    
    print(f"  Успешно: {success}, Ошибки: {fail}")

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default=DEFAULT_HOST)
    parser.add_argument("--email-host", default=DEFAULT_EMAIL_HOST)
    parser.add_argument("--count", default=100, type=int)
    parser.add_argument("--users", default=5, type=int)
    parser.add_argument("--workers", default=15, type=int)
    args = parser.parse_args()
    
    users = create_users(args.host, args.users)
    if not users:
        print("Не удалось создать пользователей")
        return
    
    # Сохраняем сессии
    with open("pref_test/test_sessions.json", "w") as f:
        json.dump([{"email": e, "cookies": c, "csrf_token": t} for e, c, t in users], f, indent=2)
    
    generate_emails(args.email_host, users, args.count, args.workers)

if __name__ == "__main__":
    main()