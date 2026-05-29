-- Опция в профиле: принимать ли анонимные письма.
-- По умолчанию выключено: безопасный дефолт, никто не получает анонимку
-- без явного согласия.
ALTER TABLE users
    ADD COLUMN accept_anonymous BOOLEAN NOT NULL DEFAULT false;

-- Флаг анонимности живёт на самом письме, а не на user_emails,
-- потому что свойство принадлежит сообщению, а не паре (user, email).
-- Это автоматически распространяется и на драфты (is_draft = true).
ALTER TABLE emails
    ADD COLUMN is_anonymous BOOLEAN NOT NULL DEFAULT false;

-- Гарантия инварианта: анонимное письмо ОБЯЗАНО иметь внутреннего отправителя.
-- Без sender_id некому "прятаться", а в support-API нечего отдавать.
ALTER TABLE emails
    ADD CONSTRAINT anonymous_requires_internal_sender
        CHECK (NOT is_anonymous OR sender_id IS NOT NULL);
