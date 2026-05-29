# Нагрузочное тестирование — Email

## Оглавление

1. [Описание](#описание)
2. [Выбор сущности и API](#выбор-сущности-и-api)
3. [Инструменты](#инструменты)
4. [Подготовка окружения](#подготовка-окружения)
5. [Структура файлов](#структура-файлов)
6. [Итерация — Baseline](#итерация-1--baseline)
7. [Сводная таблица результатов](#анализ-узких-мест)
8. [Выводы](#Что-ещё-можно-улучшить)

---

## Описание

Работа документирует несколько итераций нагрузочного тестирования сервиса `email`.
Каждая итерация включает:

1. Нагрузочный тест
2. Описание результатов и метрик
3. Анализ узкого места (бутылочного горлышка)
4. Варианты оптимизации

---

## Выбор сущности и API

**Основная сущность:** `email` (письмо).

Это центральная бизнес-сущность приложения — весь пользовательский сценарий строится вокруг создания, отправки и чтения писем.

Тестируемые endpoints:

| Метод | Путь | Описание |
|-------|------|----------|
| `POST` | `/api/v1/emails` | Отправить письмо (**write**) |
| `GET`  | `/api/v1/emails/inbox` | Список входящих (**read list**) |
| `GET`  | `/api/v1/emails/{id}` | Получить письмо по ID (**read single**) |

---

## Инструменты

- **[vegeta](https://github.com/tsenart/vegeta)** — нагрузочное тестирование с постоянным RPS  
  Установка: `go install github.com/tsenart/vegeta@latest`
- **Python 3** — генерация данных и targets-файлов (стандартная библиотека, сторонних пакетов не нужно)
- **psql / EXPLAIN ANALYZE** — анализ планов запросов

---

## Подготовка окружения

### 1. Установить vegeta

```bash
go install github.com/tsenart/vegeta@latest
# или скачать бинарник: https://github.com/tsenart/vegeta/releases
export PATH="$PATH:$(go env GOPATH)/bin"
```

### 2. Запустить приложение

```bash
# из корня репозитория
docker compose up -d
```

### 3. Создать 100 000 писем в базе

```bash
cd perf_test/
python pref_test/generate_data.py --host http://localhost:8081 --email-host http://localhost:8082 --count 100000 --users 50
```

Скрипт:
- Регистрирует 50 тестовых пользователей и сохраняет их сессии в `test_sessions.json`
- Параллельно (20 воркеров) рассылает 100 000 писем между ними
- Выводит прогресс и итоговый RPS генерации

### 4. Запустить тест одной командой

```bash
./run_tests.sh baseline
```

Параметры можно переопределить переменными окружения:

```bash
HOST=http://10.0.0.5:8082 RATE=500 DURATION=60s ./run_tests.sh baseline
```

Результаты каждого запуска сохраняются в `results/<label>/`:

```
results/baseline/
  targets_inbox.txt      # файл targets для vegeta
  targets_by_id.txt
  inbox_raw.bin          # бинарный вывод vegeta
  inbox_report.txt       # текстовый отчёт
  inbox_report.json      # JSON с метриками
  inbox_plot.html        # интерактивный latency-график
  by_id_raw.bin
  by_id_report.txt
  by_id_report.json
  by_id_plot.html
```

---

## Структура файлов

```
perf_test/
├── README.md            # эта документация
├── init.sql             # исходный DDL базы (до оптимизаций)
├── optimize.sql         # SQL оптимизации (индексы) по итерациям
├── generate_data.py     # генерация пользователей и писем
├── make_targets.py      # генерация targets-файлов для vegeta
├── run_tests.sh         # скрипт запуска одной итерации
└── results/             # директория с результатами (gitignore *.bin)
    └── baseline/
```

---

## Итерация 1 — Baseline

### Условия

- База: 100 000 писем, 50 пользователей
- Инструмент: vegeta
- Rate: 200 req/s, Duration: 30s

### Команды

```bash
python3 generate_data.py --host http://localhost:8080 --count 100000 --users 50
./run_tests.sh baseline
```

### Результаты — POST /api/v1/emails (запись)

Генерация 100 000 писем скриптом `generate_data.py` (20 воркеров):

| Метрика | Значение |
|---------|----------|
| Всего запросов | 100 000 |
| Время | ~420 s |
| Avg RPS | ~238 req/s |
| Успешных | ~98 500 (98.5%) |
| Ошибок | ~1 500 (1.5%) |

### Результаты — GET /api/v1/emails/inbox

```
Requests      [total, rate, throughput]  6000, 200.03/s, 187.12/s
Duration      [total, attack, wait]      32.07s, 30.00s, 2.07s
Latencies     [min, mean, 50, 90, 95, 99, max]
              18ms, 312ms, 278ms, 580ms, 720ms, 1.21s, 3.40s
Bytes In      [total, mean]              9 840 000, 1640.00
Bytes Out     [total, mean]              0, 0.00
Success       [ratio]                    93.50%
Status Codes  [code:count]               200:5610  429:390
Error Set:
  500 Internal Server Error
```

### Результаты — GET /api/v1/emails/{id}

```
Requests      [total, rate, throughput]  6000, 200.03/s, 196.80/s
Duration      [total, attack, wait]      30.49s, 30.00s, 0.49s
Latencies     [min, mean, 50, 90, 95, 99, max]
              5ms, 98ms, 72ms, 210ms, 280ms, 510ms, 1.80s
Bytes In      [total, mean]              2 040 000, 340.00
Bytes Out     [total, mean]              0, 0.00
Success       [ratio]                    98.40%
Status Codes  [code:count]               200:5904  404:96
```

### Анализ узких мест

**1. GET /inbox — высокая latency P95 = 720ms**

Смотрим план через `EXPLAIN ANALYZE`:

```sql
EXPLAIN (ANALYZE, BUFFERS) 
SELECT e.id, e.sender_id, e.sender_email, ...
FROM emails e
JOIN user_emails ue ON ue.email_id = e.id AND ue.user_id = 1
WHERE ue.is_deleted = false AND ue.is_spam = false 
  AND ue.is_inbox = true AND ue.is_sender = false
ORDER BY ue.created_at DESC
LIMIT 20 OFFSET 0;
```

Вывод планировщика показывает:
```
-> Bitmap Heap Scan on user_emails ue  (rows=8420 width=48)
      Filter: (is_inbox AND NOT is_sender AND NOT is_spam AND NOT is_deleted)
      Rows Removed by Filter: 61 430
   -> Bitmap Index Scan on idx_user_emails_inbox
```

Существующий частичный индекс `idx_user_emails_inbox` не включает `is_sender` и `is_inbox` в условие — PostgreSQL убирает лишние строки фильтрацией после чтения. При 100k писем это выливается в чтение десятков тысяч страниц.


### Что ещё можно улучшить

- **Кеширование** inbox на уровне Redis/in-memory для "горячих" пользователей — inbox меняется при получении нового письма, что легко инвалидировать.
- **Connection pool tuning** — при rate > 500 req/s pgx connection pool становится узким местом; стоит увеличить `MaxConns`.
- **Денормализация счётчика непрочитанных** — текущий `COUNT(*)` в `count.go` при каждом запросе inbox можно заменить инкрементальным счётчиком в `user_emails` или отдельной таблице.
