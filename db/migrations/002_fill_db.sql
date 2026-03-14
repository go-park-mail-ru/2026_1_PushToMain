TRUNCATE
    contact_category_members, contact_categories, contacts,
    folder_messages, folders, message_statuses, message_recipients, attachments,
    messages, threads,
    user_group_membership, group_permissions, permissions, user_groups,
    profile_settings, profiles, base_profiles
RESTART IDENTITY CASCADE;

-- Создание базовых профилей
INSERT INTO base_profiles (email, user_type, is_active) VALUES
    ('ivan.petrov@smail.ru', 'user', true),
    ('maria.ivanova@smail.ru', 'user', true),
    ('alexey.smirnov@smail.ru', 'user', true),
    ('elena.kozlova@smail.ru', 'user', true),
    ('dmitry.volkov@smail.ru', 'user', true),
    ('prof.sokolov@smail.ru', 'user', true),
    ('prof.popova@smail.ru', 'user', true),
    ('dean.mikhailov@smail.ru', 'admin', true),
    ('admin@smail.ru', 'admin', true),
    ('support@smail.ru', 'support', true);

-- Студенты
INSERT INTO profiles (base_profile_id, username, password_hash, first_name, last_name, middle_name, gender, department, position, group_name, auth_version) VALUES
    (1, 'ivan.petrov', '$2a$10$XKQK5QK5QK5QK5QK5QK5QO', 'Иван', 'Петров', 'Сергеевич', 'male',  'СГН', 'Студент', 'СГН3-44Б', 1),
    (2, 'maria.ivanova', '$2a$10$XKQK5QK5QK5QK5QK5QK5QO', 'Мария', 'Иванова', 'Алексеевна', 'female',  'СГН', 'Студент', 'СГН3-44Б', 1),
    (3, 'alexey.smirnov', '$2a$10$XKQK5QK5QK5QK5QK5QK5QO', 'Алексей', 'Смирнов', 'Дмитриевич', 'male',  'ИУ', 'Студент', 'ИУ6-42', 1),
    (4, 'elena.kozlov', '$2a$10$XKQK5QK5QK5QK5QK5QK5QO', 'Елена', 'Козлова', 'Павловна', 'female',  'ИУ', 'Студент', 'ИУ6-42', 1),
    (5, 'dmitry.volkov', '$2a$10$XKQK5QK5QK5QK5QK5QK5QO', 'Дмитрий', 'Волков', 'Игоревич', 'male',  'ИУ', 'Студент', 'ИУ7-69Б', 1);

INSERT INTO profiles (base_profile_id, username, password_hash, first_name, last_name, middle_name, gender, department, position, group_name, auth_version) VALUES
    (1, 'ivan.petrov', '$2a$10$XKQK5QK5QK5QK5QK5QK5QO', 'Иван', 'Петров', 'Сергеевич', 'male',  'СГН', 'Студент', 'СГН3-44Б', 1),
    (2, 'maria.ivanova', '$2a$10$XKQK5QK5QK5QK5QK5QK5QO', 'Мария', 'Иванова', 'Алексеевна', 'female',  'СГН', 'Студент', 'СГН3-44Б', 1),
    (3, 'alexey.smirnov', '$2a$10$XKQK5QK5QK5QK5QK5QK5QO', 'Алексей', 'Смирнов', 'Дмитриевич', 'male',  'ИУ', 'Студент', 'ИУ6-42', 1),
    (4, 'elena.kozlov', '$2a$10$XKQK5QK5QK5QK5QK5QK5QO', 'Елена', 'Козлова', 'Павловна', 'female',  'ИУ', 'Студент', 'ИУ6-42', 1),
    (5, 'dmitry.volkov', '$2a$10$XKQK5QK5QK5QK5QK5QK5QO', 'Дмитрий', 'Волков', 'Игоревич', 'male',  'ИУ', 'Студент', 'ИУ7-69Б', 1);

-- Преподаватели и сотрудники
INSERT INTO profiles (base_profile_id, username, password_hash, first_name, last_name, middle_name, gender, department, position, auth_version) VALUES
    (6, 'prof.sokolov', '$2a$10$XKQK5QK5QK5QK5QK5QK5QO', 'Александр', 'Соколов', 'Петрович', 'male',  'ИУ', 'Профессор, д.т.н.', 1),
    (7, 'prof.popova', '$2a$10$XKQK5QK5QK5QK5QK5QK5QO', 'Екатерина', 'Попова', 'Владимировна', 'female',  'ИУ', 'Доцент, к.ф.н.', 1),
    (8, 'dean.mikhailov', '$2a$10$XKQK5QK5QK5QK5QK5QK5QO', 'Михаил', 'Михайлов', 'Сергеевич', 'male',  'Деканат', 'Декан факультета', 1),
    (9, 'admin', '$2a$10$XKQK5QK5QK5QK5QK5QK5QO', 'Админ', 'Админов', NULL, 'male',  'Администрация', 'Системный администратор', 1),
    (10, 'support', '$2a$10$XKQK5QK5QK5QK5QK5QK5QO', 'Служба', 'Поддержки', NULL, 'female',  'Поддержка', 'Специалист поддержки', 1);

--- Создание групп
INSERT INTO user_groups (group_code, group_name, description, group_type, priority, is_system) VALUES
    ('all_students', 'Все студенты', 'Все студенты университета', 'system', 10, true),
    ('all_teachers', 'Все преподаватели', 'Все преподаватели и сотрудники', 'system', 10, true),
    ('sgn3_44b', 'СГН3-44Б', 'СГН3-44Б', 'custom', 20, false),
    ('iu6_42', 'ИУ6-42', 'Студенты группы ИУ6-42', 'custom', 20, false),
    ('iu7_69b', 'ИУ7-69Б', 'Студенты группы ИУ7-69Б', 'custom', 20, false),
    ('iu', 'Факультет ИУ', 'Преподаватели факультета ИУ', 'custom', 15, false),
    ('deans_office', 'Деканат', 'Сотрудники деканата', 'custom', 5, false),
    ('admins', 'Администраторы', 'Администраторы системы', 'system', 1, true),
    ('support_team', 'Поддержка', 'Команда поддержки', 'system', 2, true);

-- Студенты
INSERT INTO user_group_membership (profile_id, group_id, assigned_by, is_active) VALUES
    (1, 1, 9, true),
    (2, 1, 9, true),
    (3, 1, 9, true),
    (4, 1, 9, true),
    (5, 1, 9, true),
    (1, 3, 9, true),
    (2, 3, 9, true),
    (3, 4, 9, true),
    (4, 4, 9, true),
    (5, 5, 9, true);

-- Преподаватели и сотрудники
INSERT INTO user_group_membership (profile_id, group_id, assigned_by, is_active) VALUES
    (6, 2, 9, true),
    (7, 2, 9, true),
    (8, 2, 9, true),
    (6, 6, 9, true),
    (7, 6, 9, true),
    (8, 7, 9, true),
    (9, 8, 9, true),
    (10, 9, 9, true);

--- Создание разрешений ---
INSERT INTO permissions (permission_code, permission_name, resource_type, action) VALUES
    -- Базовые разрешения для студентов
    ('student:message:read', 'Чтение сообщений', 'messages', 'read'),
    ('student:message:send', 'Отправка сообщений', 'messages', 'create'),
    ('student:contacts:manage', 'Управление контактами', 'contacts', 'manage'),
    ('student:folders:manage', 'Управление папками', 'folders', 'manage'),

    -- Для преподавателей
    ('teacher:message:read_all', 'Чтение всех сообщений', 'messages', 'read'),
    ('teacher:broadcast:send', 'Массовая рассылка', 'messages', 'create'),

    -- Для администраторов
    ('admin:users:manage', 'Управление пользователями', 'users', 'manage'),
    ('admin:settings:edit', 'Редактирование настроек', 'settings', 'update'),
    ('admin:all:access', 'Полный доступ', 'admin_panel', 'manage');

--- Назначение разрешений группам ---
-- Студенты - базовые разрешения
INSERT INTO group_permissions (group_id, permission_id) VALUES
    (1, 1), (1, 2), (1, 3), (1, 4);

-- Преподаватели - больше разрешений
INSERT INTO group_permissions (group_id, permission_id) VALUES
    (2, 1), (2, 2), (2, 3), (2, 4), (2, 5), (2, 6);

-- Администраторы - все разрешения
INSERT INTO group_permissions (group_id, permission_id) VALUES
    (8, 1), (8, 2), (8, 3), (8, 4), (8, 5), (8, 6), (8, 7), (8, 8), (8, 9);

-- Поддержка - нужные разрешения
INSERT INTO group_permissions (group_id, permission_id) VALUES
    (9, 1), (9, 5), (9, 7);

--- Настройки пользователей ---
INSERT INTO profile_settings (profile_id, language, notifications_email, notifications_push, messages_per_page) VALUES
    (1, 'ru',  true, true, 50),
    (2, 'ru', true, false, 100),
    (3, 'en',  false, true, 50),
    (4, 'ru',  true, true, 25),
    (5, 'ru', true, true, 50),
    (6, 'ru',  true, false, 50),
    (7, 'ru',  true, true, 50),
    (8, 'ru',  false, false, 100),
    (9, 'en', true, true, 50),
    (10, 'ru', true, true, 50);

--- Создание системных папок для каждого пользователя
INSERT INTO folders (profile_id, name, type, system_name, sort_order)
SELECT
    p.id,
    CASE system_name
        WHEN 'inbox' THEN 'Входящие'
        WHEN 'sent' THEN 'Отправленные'
        WHEN 'drafts' THEN 'Черновики'
        WHEN 'trash' THEN 'Корзина'
        WHEN 'spam' THEN 'Спам'
    END,
    'system',
    system_name,
    sort_order
FROM profiles p
CROSS JOIN (
    VALUES ('inbox', 1), ('sent', 2), ('drafts', 3), ('trash', 4), ('spam', 5)
) AS system_folders(system_name, sort_order)
WHERE p.id <= 10;

--- Создание контактов между пользователями ---
-- Иван добавляет одногруппников
INSERT INTO contacts (profile_id, contact_email, first_name, last_name, contact_type, is_favorite) VALUES
    (1, 'maria.ivanova@smail.ru', 'Мария', 'Иванова', 'internal', true),
    (1, 'alexey.smirnov@smail.ru', 'Алексей', 'Смирнов', 'internal', false),
    (1, 'prof.sokolov@smail.ru', 'Александр', 'Соколов', 'internal', true);

-- Мария добавляет контакты
INSERT INTO contacts (profile_id, contact_email, first_name, last_name, contact_type, is_favorite) VALUES
    (2, 'ivan.petrov@smail.ru', 'Иван', 'Петров', 'internal', true),
    (2, 'elena.kozlov@smail.ru', 'Елена', 'Козлова', 'internal', true);

--- Создание категорий контактов ---
INSERT INTO contact_categories (profile_id, name, type, color, sort_order) VALUES
    (1, 'Одногруппники', 'user_defined', '#FF6B6B', 1),
    (1, 'Преподаватели', 'user_defined', '#4ECDC4', 2),
    (2, 'Друзья', 'user_defined', '#45B7D1', 1),
    (2, 'Учеба', 'user_defined', '#96CEB4', 2);

--- Добавление контактов в категории
INSERT INTO contact_category_members (category_id, contact_id)
SELECT
    c.id as category_id,
    ct.id as contact_id
FROM contact_categories c
JOIN contacts ct ON ct.profile_id = c.profile_id
WHERE c.profile_id = 1
  AND c.name = 'Одногруппники'
  AND ct.contact_email IN ('maria.ivanova@smail.ru', 'alexey.smirnov@smail.ru');

INSERT INTO contact_category_members (category_id, contact_id)
SELECT
    c.id as category_id,
    ct.id as contact_id
FROM contact_categories c
JOIN contacts ct ON ct.profile_id = c.profile_id
WHERE c.profile_id = 1
  AND c.name = 'Преподаватели'
  AND ct.contact_email = 'prof.sokolov@smail.ru';

--- Проверка созданных данных
SELECT 'base_profiles' as table_name, COUNT(*) as count FROM base_profiles
UNION ALL
SELECT 'profiles', COUNT(*) FROM profiles
UNION ALL
SELECT 'user_groups', COUNT(*) FROM user_groups
UNION ALL
SELECT 'user_group_membership', COUNT(*) FROM user_group_membership
UNION ALL
SELECT 'permissions', COUNT(*) FROM permissions
UNION ALL
SELECT 'group_permissions', COUNT(*) FROM group_permissions
UNION ALL
SELECT 'profile_settings', COUNT(*) FROM profile_settings
UNION ALL
SELECT 'folders', COUNT(*) FROM folders
UNION ALL
SELECT 'contacts', COUNT(*) FROM contacts
UNION ALL
SELECT 'contact_categories', COUNT(*) FROM contact_categories
ORDER BY table_name;

-- Показать созданных пользователей с их группами
SELECT
    p.last_name || ' ' || p.first_name as full_name,
    p.department,
    p.position,
    p.group_name as student_group,
    string_agg(DISTINCT ug.group_name, ', ') as permission_groups
FROM profiles p
LEFT JOIN user_group_membership ugm ON p.id = ugm.profile_id
LEFT JOIN user_groups ug ON ugm.group_id = ug.id
GROUP BY p.id, p.last_name, p.first_name, p.department, p.position, p.group_name
ORDER BY p.position NULLS LAST, p.last_name;