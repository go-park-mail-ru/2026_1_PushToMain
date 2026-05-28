-- =============================================================================
-- perf_test/optimize.sql
-- Оптимизации БД после первой итерации нагрузочного тестирования.
-- Применять последовательно; каждая секция — отдельная итерация.
-- =============================================================================

-- =============================================================================
-- ИТЕРАЦИЯ 2: Добавление составных индексов для queryUserMailbox
-- =============================================================================

-- Проблема: в queryUserMailbox и GetSentEmails запрос фильтрует user_emails по
-- (user_id, is_deleted, is_spam, is_inbox, is_sender) и сортирует по created_at.
-- Существующий idx_user_emails_inbox покрывает только inbox, но не sent и не
-- произвольные комбинации флагов. PostgreSQL вынужден делать Seq Scan или
-- Index Scan с дополнительной фильтрацией.

-- Индекс для "входящих" (заменяет partial index, добавляем is_sender):
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ue_inbox_v2
    ON user_emails (user_id, created_at DESC)
    WHERE is_deleted = false AND is_spam = false
      AND is_inbox = true AND is_sender = false;

-- Индекс для отправленных:
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ue_sent
    ON user_emails (user_id, created_at DESC)
    WHERE is_sender = true AND is_deleted = false AND is_draft = false;

-- Примечание: столбца is_draft нет в user_emails — фильтр is_draft = false
-- применяется на таблице emails. Поэтому для GetSentEmails составной индекс
-- включает JOIN. Вместо этого добавим индекс на emails(sender_id, is_draft):
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_emails_sender_not_draft
    ON emails (sender_id, created_at DESC)
    WHERE is_draft = false;

-- =============================================================================
-- ИТЕРАЦИЯ 2: Индекс для GetEmailByID (коррелированный подзапрос recipients)
-- =============================================================================

-- Проблема: в GetEmailByID и queryUserMailbox используется коррелированный
-- подзапрос:
--   (SELECT string_agg(er.recipient_email, ',')
--    FROM email_recipients er WHERE er.email_id = e.id)
--
-- Индекс idx_email_recipients_email_id уже существует, но покрывающий индекс
-- (включающий recipient_email) позволит PostgreSQL обойтись без обращения к
-- heap (Index Only Scan).

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_er_email_id_covering
    ON email_recipients (email_id)
    INCLUDE (recipient_email);

-- =============================================================================
-- ИТЕРАЦИЯ 3: Покрывающий индекс для user_emails (частый read-path)
-- =============================================================================

-- queryUserMailbox читает из user_emails столбцы:
--   is_read, is_starred, is_spam, is_deleted, created_at, is_inbox, is_sender
-- Покрывающий индекс позволяет избежать обращений к heap-странице.

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ue_covering_inbox
    ON user_emails (user_id, created_at DESC)
    INCLUDE (email_id, is_read, is_starred, is_spam, is_deleted, is_inbox, is_sender)
    WHERE is_deleted = false AND is_spam = false;

-- =============================================================================
-- ИТЕРАЦИЯ 3: Устранение проблемы GetAllEmails (двойной запрос + sort в Go)
-- =============================================================================
-- GetAllEmails делает два отдельных SELECT (inbox + sent) и сортирует в памяти.
-- Это плохо масштабируется при большом объёме данных.
-- Рекомендуется заменить на единый SQL-запрос (см. README, раздел Оптимизации).

-- =============================================================================
-- ИТЕРАЦИЯ 4: VACUUM / ANALYZE после массовой вставки 100k писем
-- =============================================================================

ANALYZE emails;
ANALYZE email_recipients;
ANALYZE user_emails;
