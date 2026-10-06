-- 1. Таблица государственного реестра сотрудников
CREATE TABLE digital_employees (
    id BIGSERIAL PRIMARY KEY,
    user_id INT NOT NULL UNIQUE,          
    email VARCHAR(255) NOT NULL UNIQUE,   
    inn VARCHAR(12) NOT NULL UNIQUE,      
    passport_number VARCHAR(20) NOT NULL, 
    insurance_number VARCHAR(11) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    gov_status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE', -- Спецификация: ACTIVE или BLOCKED
    digital_signature_status VARCHAR(50) NOT NULL DEFAULT 'VALID'
);

-- 2. Таблица системных мастер-токенов доступа госсервиса
CREATE TABLE service_access_tokens (
    id SERIAL PRIMARY KEY,
    token_value TEXT NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. Индексы для ускорения SOAP поиска
CREATE INDEX idx_gov_employees_email ON digital_employees(email);
CREATE INDEX idx_gov_tokens_value ON service_access_tokens(token_value);

-- =========================================================================
-- НАПОЛНЕНИЕ ТЕСТОВЫМИ ДАННЫМИ ПОД ВАШИ СЦЕНАРИИ
-- =========================================================================

-- Мастер-токен для авторизации запросов самого auth-service
INSERT INTO service_access_tokens (token_value, is_active) VALUES
('gos_master_secure_token_2026', true);

-- Базовый мобильный пользователь холдинга (Всё отлично)
INSERT INTO digital_employees (user_id, email, inn, passport_number, insurance_number, first_name, last_name, gov_status) VALUES
(10001, 'mobile.user@example.com', '770112345678', '4512 987654', '12345678901', 'Иван', 'Петров', 'ACTIVE');

-- Твины 1, 4 и все остальные до 500 — ХОРОШИЕ (Доступ разрешен, статус ACTIVE)
INSERT INTO digital_employees (user_id, email, inn, passport_number, insurance_number, first_name, last_name, gov_status) VALUES
(20001, 'twin.1@example.com', '770200000001', '4515 000001', '90000000001', 'Первый', 'ХорошийТвин', 'ACTIVE');

-- twin.2@example.com — СЦЕНАРИЙ БЛОКИРОВКИ (Запись в реестре есть, но статус BLOCKED)
INSERT INTO digital_employees (user_id, email, inn, passport_number, insurance_number, first_name, last_name, gov_status) VALUES
(20002, 'twin.2@example.com', '770200000002', '4515 000002', '90000000002', 'Заблокированный', 'ВторойТвин', 'BLOCKED');

-- twin.3@example.com — СЦЕНАРИЙ ОТСУТСТВИЯ В РЕЕСТРЕ
-- Мы его просто НЕ добавляем в эту таблицу. При поиске SOAP выдаст ошибку, что гражданин отсутствует в Гос-БД.

-- Генерация остальных хороших активных твинов со 4 по 500
INSERT INTO digital_employees (user_id, email, inn, passport_number, insurance_number, first_name, last_name, gov_status)
SELECT 
    30000 + i AS user_id,
    'twin.' || i || '@example.com' AS email,
    '7703' || LPAD(i::text, 8, '0') AS inn,
    '4520 ' || LPAD(i::text, 6, '0') AS passport_number,
    '200' || LPAD(i::text, 8, '0') AS insurance_number,
    'Робот_' || i AS first_name,
    'Твинов' AS last_name,
    'ACTIVE' AS gov_status
FROM generate_series(4, 500) AS i;