-- ============================================
-- 1. СОЗДАНИЕ ПОЛЬЗОВАТЕЛЕЙ ДЛЯ МИКРОСЕРВИСОВ
-- ============================================

-- Сначала создадим базу данных (если ещё нет)
-- CREATE DATABASE smail_db OWNER postgres;

-- Подключаемся к базе
\c smail_db

-- Удаляем старых пользователей (если есть)
DROP OWNED BY user_service CASCADE;
DROP OWNED BY email_service CASCADE;
DROP OWNED BY folder_service CASCADE;
DROP USER IF EXISTS user_service;
DROP USER IF EXISTS email_service;
DROP USER IF EXISTS folder_service;
DROP USER IF EXISTS monitor_user;

-- Создаём пользователей с разными паролями
CREATE USER user_service WITH PASSWORD 'user_strong_pass_123' LOGIN;
CREATE USER email_service WITH PASSWORD 'email_strong_pass_456' LOGIN;
CREATE USER folder_service WITH PASSWORD 'folder_strong_pass_789' LOGIN;
CREATE USER monitor_user WITH PASSWORD 'monitor_pass_000' LOGIN;

-- ============================================
-- 2. НАСТРОЙКА ПРАВ ДОСТУПА
-- ============================================

-- Даём права на подключение к базе
GRANT CONNECT ON DATABASE smail_db TO user_service, email_service, folder_service, monitor_user;

-- Даём права на схему public
GRANT USAGE ON SCHEMA public TO user_service, email_service, folder_service, monitor_user;

-- ============================================
-- 3. ПРАВА ДЛЯ USER SERVICE (только пользователи)
-- ============================================

-- Может читать/писать в таблицу users
GRANT SELECT, INSERT, UPDATE, DELETE ON users TO user_service;
GRANT USAGE, SELECT ON SEQUENCE users_id_seq TO user_service;

-- Доступ к пользователям в других целях (только чтение)
GRANT SELECT ON users TO email_service, folder_service;

-- ============================================
-- 4. ПРАВА ДЛЯ EMAIL SERVICE (письма, вложения, user_emails)
-- ============================================

-- emails
GRANT SELECT, INSERT, UPDATE, DELETE ON emails TO email_service;
GRANT USAGE, SELECT ON SEQUENCE emails_id_seq TO email_service;

-- attachments
GRANT SELECT, INSERT, UPDATE, DELETE ON attachments TO email_service;
GRANT USAGE, SELECT ON SEQUENCE attachments_id_seq TO email_service;

-- user_emails
GRANT SELECT, INSERT, UPDATE, DELETE ON user_emails TO email_service;
GRANT USAGE, SELECT ON SEQUENCE user_emails_id_seq TO email_service;

-- draft_receivers
GRANT SELECT, INSERT, UPDATE, DELETE ON draft_receivers TO email_service;
GRANT USAGE, SELECT ON SEQUENCE draft_receivers_id_seq TO email_service;

-- ============================================
-- 5. ПРАВА ДЛЯ FOLDER SERVICE
-- ============================================

-- folders
GRANT SELECT, INSERT, UPDATE, DELETE ON folders TO folder_service;
GRANT USAGE, SELECT ON SEQUENCE folders_id_seq TO folder_service;

-- folder_emails
GRANT SELECT, INSERT, DELETE ON folder_emails TO folder_service;
GRANT USAGE, SELECT ON SEQUENCE folder_emails_id_seq TO folder_service;

-- ============================================
-- 6. ПРАВА ДЛЯ MONITOR USER (только чтение для мониторинга)
-- ============================================

GRANT SELECT ON ALL TABLES IN SCHEMA public TO monitor_user;
GRANT USAGE ON SCHEMA public TO monitor_user;

-- ============================================
-- 7. НАСТРОЙКА БЕЗОПАСНОСТИ
-- ============================================

-- Отключаем публичный доступ по умолчанию
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON DATABASE smail_db FROM PUBLIC;

-- Устанавливаем безопасные настройки для новых таблиц
ALTER DEFAULT PRIVILEGES IN SCHEMA public 
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO user_service, email_service, folder_service;
ALTER DEFAULT PRIVILEGES IN SCHEMA public 
    GRANT USAGE, SELECT ON SEQUENCES TO user_service, email_service, folder_service;