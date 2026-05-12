CREATE TYPE order_status AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');

CREATE TABLE orders (
    pk SERIAL PRIMARY KEY,
    id TEXT UNIQUE NOT NULL,
    user_id BIGINT NOT NULL,
    status order_status NOT NULL,
    accrual BIGINT,
    added_at TIMESTAMPTZ NOT NULL
);