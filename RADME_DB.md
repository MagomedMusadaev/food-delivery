-- Таблица пользователей
-- Хранит информацию о зарегистрированных пользователях
    CREATE TABLE users (
        id SERIAL PRIMARY KEY, -- Уникальный идентификатор пользователя
        firstname VARCHAR(255) NOT NULL, -- Имя пользователя
        email VARCHAR(255) UNIQUE NOT NULL, -- Уникальный email пользователя
        password_hash TEXT NOT NULL, -- Хэшированный пароль пользователя
        phone VARCHAR(50) UNIQUE NOT NULL, -- Уникальный номер телефона пользователя
        created_at TIMESTAMP DEFAULT NOW(), -- Дата и время создания записи
        status VARCHAR(20) DEFAULT 'active' -- Статус пользователя: active, suspended, blocked и т.д.
    );

-- Таблица токенов (для хранения refresh-токенов)
-- Используется для хранения информации о токенах пользователей
    CREATE TABLE tokens (
        id SERIAL PRIMARY KEY,              -- Уникальный идентификатор токена
        user_id INT REFERENCES users(id),   -- Ссылка на ID пользователя из таблицы users
        token TEXT NOT NULL,                -- Токен (refresh-token)
        expires_at TIMESTAMP NOT NULL,      -- Дата и время истечения токена
        created_at TIMESTAMP DEFAULT NOW(), -- Дата и время создания записи
        CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE -- Ссылка на пользователя, удаление токенов при удалении пользователя
    );









-- Таблица истории входов
-- Хранит информацию о всех входах пользователей в систему
    CREATE TABLE login_history (
        id SERIAL PRIMARY KEY, -- Уникальный идентификатор записи
        user_id INT REFERENCES users(id), -- Ссылка на ID пользователя из таблицы users
        login_time TIMESTAMP DEFAULT NOW(), -- Дата и время входа пользователя
        ip_address INET -- IP-адрес, с которого выполнен вход (тип INET для IP-адресов)
    );
