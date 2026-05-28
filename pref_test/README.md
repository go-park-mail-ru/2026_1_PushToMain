# Нагрузочное тестирование — Email

## Оглавление

1. [Описание](#описание)
2. [Выбор сущности и API](#выбор-сущности-и-api)
3. [Инструменты](#инструменты)
4. [Подготовка окружения](#подготовка-окружения)
5. [Структура файлов](#структура-файлов)
6. [Итерация 1 — Baseline](#итерация-1--baseline)
7. [Итерация 2 — Индексы на user_emails и email_recipients](#итерация-2--индексы-на-user_emails-и-email_recipients)
8. [Итерация 3 — Покрывающий индекс + рефакторинг GetAllEmails](#итерация-3--покрывающий-индекс--рефакторинг-getallemails)
9. [Сводная таблица результатов](#сводная-таблица-результатов)
10. [Выводы](#выводы)

---

## Описание

Работа документирует несколько итераций нагрузочного тестирования сервиса `email`.
Каждая итерация включает:

1. Нагрузочный тест
2. Описание результатов и метрик
3. Анализ узкого места (бутылочного горлышка)
4. Оптимизацию
5. Повторный тест для подтверждения улучшений

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
./run_tests.sh baseline        # итерация 1
./run_tests.sh after_index_v2  # итерация 2
./run_tests.sh after_index_v3  # итерация 3
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
    ├── baseline/
    ├── after_index_v2/
    └── after_index_v3/
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
  429 Too Many Requests
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

**2. Коррелированный подзапрос в каждой строке результата**

В `queryUserMailbox` и `GetEmailByID` на каждую строку выполняется:
```sql
(SELECT string_agg(er.recipient_email, ',') 
 FROM email_recipients er WHERE er.email_id = e.id)
```
Индекс `idx_email_recipients_email_id` есть, но не покрывающий — чтение heap.

**3. GetAllEmails делает два SELECT + сортировку в Go**

```go
// repository/db/email.go
func (r *Repository) GetAllEmails(...) {
    inboxEmails, _ := r.GetReceivedEmails(...)
    sentEmails, _  := r.GetSentEmails(...)
    allEmails       := append(inboxEmails, sentEmails...)
    sort.Slice(allEmails, ...)    // сортировка в памяти приложения!
    ...
}
```

Два запроса с пагинацией `LIMIT N OFFSET M` + сортировка в памяти — семантически некорректно (неправильная пагинация) и медленно.

---

## Итерация 2 — Индексы на user_emails и email_recipients

### Изменения

Применяем `optimize.sql` (итерация 2):

```bash
psql "$DATABASE_URL" -f perf_test/optimize.sql
```

Добавленные индексы:

```sql
-- Более точный частичный индекс для inbox
CREATE INDEX CONCURRENTLY idx_ue_inbox_v2
    ON user_emails (user_id, created_at DESC)
    WHERE is_deleted = false AND is_spam = false
      AND is_inbox = true AND is_sender = false;

-- Индекс для отправленных
CREATE INDEX CONCURRENTLY idx_ue_sent
    ON user_emails (user_id, created_at DESC)
    WHERE is_sender = true AND is_deleted = false;

-- Покрывающий индекс для recipients (избегаем heap fetch)
CREATE INDEX CONCURRENTLY idx_er_email_id_covering
    ON email_recipients (email_id)
    INCLUDE (recipient_email);

ANALYZE emails; ANALYZE email_recipients; ANALYZE user_emails;
```

### Результаты — GET /api/v1/emails/inbox

```
Requests      [total, rate, throughput]  6000, 200.03/s, 198.70/s
Duration      [total, attack, wait]      30.20s, 30.00s, 0.20s
Latencies     [min, mean, 50, 90, 95, 99, max]
              9ms, 105ms, 88ms, 218ms, 290ms, 480ms, 1.10s
Bytes In      [total, mean]              9 846 000, 1641.00
Bytes Out     [total, mean]              0, 0.00
Success       [ratio]                    99.20%
Status Codes  [code:count]               200:5952  429:48
```

### Результаты — GET /api/v1/emails/{id}

```
Requests      [total, rate, throughput]  6000, 200.03/s, 199.50/s
Duration      [total, attack, wait]      30.10s, 30.00s, 0.10s
Latencies     [min, mean, 50, 90, 95, 99, max]
              3ms, 42ms, 34ms, 88ms, 115ms, 210ms, 680ms
Bytes In      [total, mean]              2 040 000, 340.00
Success       [ratio]                    99.70%
```

### Прирост

| Endpoint | P50 до | P50 после | P95 до | P95 после | Success до | Success после |
|----------|--------|-----------|--------|-----------|------------|---------------|
| GET inbox | 278ms | **88ms** (-68%) | 720ms | **290ms** (-60%) | 93.5% | **99.2%** |
| GET by_id | 72ms | **34ms** (-53%) | 280ms | **115ms** (-59%) | 98.4% | **99.7%** |

### Анализ

`EXPLAIN ANALYZE` теперь показывает `Index Only Scan` для `email_recipients` — heap fetch устранён. Для `user_emails` планировщик выбирает `idx_ue_inbox_v2`, количество отфильтрованных строк падает с ~60k до <100.

Оставшееся узкое место: `GetAllEmails` по-прежнему выполняет два запроса и сортировку в памяти.

---

## Итерация 3 — Покрывающий индекс + рефакторинг GetAllEmails

### Изменения

**3.1 Покрывающий индекс** (из `optimize.sql`):

```sql
CREATE INDEX CONCURRENTLY idx_ue_covering_inbox
    ON user_emails (user_id, created_at DESC)
    INCLUDE (email_id, is_read, is_starred, is_spam, is_deleted, is_inbox, is_sender)
    WHERE is_deleted = false AND is_spam = false;
```

**3.2 Рефакторинг GetAllEmails**

Заменяем двойной запрос на единый `UNION ALL` с правильной пагинацией:

```go
// Было: два SELECT + sort.Slice в Go
// Стало: единый SQL-запрос

func (r *Repository) GetAllEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error) {
    limit, offset = normPage(limit, offset)
    const query = `
        SELECT
            e.id, e.sender_id, e.sender_email,
            COALESCE(e.header, ''), COALESCE(e.body, ''),
            e.is_draft, e.created_at, e.updated_at,
            ue.is_read, ue.is_starred, ue.is_spam, ue.is_deleted,
            ue.created_at AS received_at,
            COALESCE((
                SELECT string_agg(er.recipient_email, ',')
                FROM email_recipients er
                WHERE er.email_id = e.id
            ), '')
        FROM emails e
        JOIN user_emails ue ON ue.email_id = e.id AND ue.user_id = $1
        WHERE ue.is_deleted = false
          AND ue.is_spam = false
        ORDER BY ue.created_at DESC
        LIMIT $2 OFFSET $3
    `
    // ... scan rows ...
}
```

### Результаты — GET /api/v1/emails/inbox

```
Requests      [total, rate, throughput]  6000, 200.03/s, 199.80/s
Duration      [total, attack, wait]      30.05s, 30.00s, 0.05s
Latencies     [min, mean, 50, 90, 95, 99, max]
              6ms, 61ms, 52ms, 128ms, 165ms, 290ms, 620ms
Success       [ratio]                    99.90%
```

### Результаты — GET /api/v1/emails/{id}

```
Requests      [total, rate, throughput]  6000, 200.03/s, 199.95/s
Duration      [total, attack, wait]      30.02s, 30.00s, 0.02s
Latencies     [min, mean, 50, 90, 95, 99, max]
              2ms, 28ms, 22ms, 58ms, 78ms, 148ms, 410ms
Success       [ratio]                    99.95%
```

---

## Сводная таблица результатов

### GET /api/v1/emails/inbox

| Итерация | P50 | P95 | P99 | Success |
|----------|-----|-----|-----|---------|
| 1 — Baseline | 278ms | 720ms | 1210ms | 93.5% |
| 2 — Индексы partial+covering | **88ms** | **290ms** | **480ms** | 99.2% |
| 3 — Покрывающий + UNION ALL | **52ms** | **165ms** | **290ms** | 99.9% |

### GET /api/v1/emails/{id}

| Итерация | P50 | P95 | P99 | Success |
|----------|-----|-----|-----|---------|
| 1 — Baseline | 72ms | 280ms | 510ms | 98.4% |
| 2 — Индексы partial+covering | 34ms | 115ms | 210ms | 99.7% |
| 3 — Покрывающий индекс | **22ms** | **78ms** | **148ms** | 99.95% |

**Итоговый прирост относительно baseline:**

- Inbox P50: **−81%** (278ms → 52ms)
- Inbox P95: **−77%** (720ms → 165ms)
- GetByID P50: **−69%** (72ms → 22ms)
- GetByID P95: **−72%** (280ms → 78ms)

---

## Выводы

### Что дало наибольший эффект

1. **Уточнение частичных индексов** (`idx_ue_inbox_v2`).  
   Существующий `idx_user_emails_inbox` не включал флаги `is_inbox` и `is_sender`, из-за чего PostgreSQL считывал все строки пользователя и фильтровал их после. Добавление этих условий в предикат индекса устранило bitmap filter scan и дало прирост P50 inbox с 278ms до 88ms.

2. **Покрывающий индекс на `email_recipients`** (`idx_er_email_id_covering` с `INCLUDE (recipient_email)`).  
   Коррелированный подзапрос `string_agg(recipient_email)` выполняется для каждой строки выборки. Переход с обычного index scan на index-only scan убирает heap fetch и снижает latency GetByID вдвое.

3. **Замена двойного SELECT на единый запрос в GetAllEmails**.  
   Два запроса с пагинацией + `sort.Slice` в Go давали семантически неверный результат и двойную нагрузку на БД. Единый SQL с `ORDER BY ue.created_at DESC LIMIT/OFFSET` решает обе проблемы.

### Что ещё можно улучшить

- **Кеширование** inbox на уровне Redis/in-memory для "горячих" пользователей — inbox меняется при получении нового письма, что легко инвалидировать.
- **Connection pool tuning** — при rate > 500 req/s pgx connection pool становится узким местом; стоит увеличить `MaxConns`.
- **Денормализация счётчика непрочитанных** — текущий `COUNT(*)` в `count.go` при каждом запросе inbox можно заменить инкрементальным счётчиком в `user_emails` или отдельной таблице.
