CREATE TABLE users (
    user_id BIGSERIAL PRIMARY KEY,
    login TEXT UNIQUE NOT NULL,
    pwd_hash TEXT NOT NULL
);