CREATE TABLE balances (
    pk BIGSERIAL PRIMARY KEY,
    user_id BIGINT UNIQUE NOT NULL,
    current BIGINT NOT NULL DEFAULT 0 CONSTRAINT chk_balances_not_negative CHECK (current >= 0),
    withdrawn BIGINT NOT NULL DEFAULT 0
);