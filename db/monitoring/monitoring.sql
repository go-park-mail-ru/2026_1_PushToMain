-- ============================================
-- 1. МОНИТОРИНГ АКТИВНЫХ СОЕДИНЕНИЙ
-- ============================================
-- Текущие соединения по пользователям
SELECT 
    usename AS username,
    count(*) AS connections,
    state,
    application_name
FROM pg_stat_activity
WHERE datname = 'smail_db'
GROUP BY usename, state, application_name
ORDER BY connections DESC;

-- Детальная информация о соединениях
SELECT 
    pid,
    usename,
    application_name,
    client_addr,
    state,
    now() - query_start AS duration,
    query
FROM pg_stat_activity
WHERE datname = 'smail_db'
ORDER BY duration DESC NULLS LAST;

-- ============================================
-- 2. МОНИТОРИНГ БЛОКИРОВОК
-- ============================================
-- Текущие блокировки
SELECT 
    blocked_locks.pid AS blocked_pid,
    blocked_activity.query AS blocked_query,
    blocking_locks.pid AS blocking_pid,
    blocking_activity.query AS blocking_query
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
    AND blocking_locks.database IS NOT DISTINCT FROM blocked_locks.database
    AND blocking_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
    AND blocking_locks.page IS NOT DISTINCT FROM blocked_locks.page
    AND blocking_locks.tuple IS NOT DISTINCT FROM blocked_locks.tuple
    AND blocking_locks.virtualxid IS NOT DISTINCT FROM blocked_locks.virtualxid
    AND blocking_locks.transactionid IS NOT DISTINCT FROM blocked_locks.transactionid
JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted;

-- Длительные транзакции (> 5 минут)
SELECT 
    pid,
    usename,
    now() - xact_start AS duration,
    state,
    query
FROM pg_stat_activity
WHERE now() - xact_start > interval '5 minutes'
    AND state != 'idle'
ORDER BY duration DESC;

-- ============================================
-- 3. МОНИТОРИНГ ПРОИЗВОДИТЕЛЬНОСТИ
-- ============================================
-- Самые медленные запросы (по времени выполнения)
SELECT 
    query,
    calls,
    total_time / 1000 AS total_seconds,
    mean_time AS avg_ms,
    max_time AS max_ms
FROM pg_stat_statements
WHERE dbid = (SELECT oid FROM pg_database WHERE datname = 'smail_db')
ORDER BY mean_time DESC
LIMIT 20;

-- Самые частые запросы
SELECT 
    query,
    calls,
    total_time / 1000 AS total_seconds,
    mean_time AS avg_ms
FROM pg_stat_statements
WHERE dbid = (SELECT oid FROM pg_database WHERE datname = 'smail_db')
ORDER BY calls DESC
LIMIT 20;

-- Размеры таблиц
SELECT 
    relname AS table_name,
    pg_size_pretty(pg_total_relation_size(relid)) AS total_size,
    pg_size_pretty(pg_relation_size(relid)) AS data_size,
    pg_size_pretty(pg_total_relation_size(relid) - pg_relation_size(relid)) AS index_size
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC;

-- Размеры индексов
SELECT 
    indexrelname AS index_name,
    tablename AS table_name,
    pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_catalog.pg_stat_user_indexes
ORDER BY pg_relation_size(indexrelid) DESC;

-- ============================================
-- 4. МОНИТОРИНГ НАГРУЗКИ ДИСКА/ПАМЯТИ
-- ============================================
-- Частота сканирований таблиц
SELECT 
    relname AS table_name,
    seq_scan AS sequential_scans,
    idx_scan AS index_scans,
    n_live_tup AS rows
FROM pg_stat_user_tables
WHERE seq_scan > 0
ORDER BY seq_scan DESC;

-- Hit ratio кэша (должен быть > 0.99)
SELECT 
    sum(heap_blks_hit) / nullif(sum(heap_blks_hit) + sum(heap_blks_read), 0) AS cache_hit_ratio
FROM pg_statio_user_tables;
