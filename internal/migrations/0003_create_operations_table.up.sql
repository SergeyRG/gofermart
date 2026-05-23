CREATE TYPE operation_type AS ENUM ('WITHDRAW', 'DEPOSIT');

CREATE TABLE operations (
    pk BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    order_id TEXT UNIQUE NOT NULL,
    op_type operation_type NOT NULL,
    sum BIGINT NOT NULL CONSTRAINT chk_oper_sum_greater_zero CHECK (sum > 0),
    processed_at TIMESTAMPTZ NOT NULL
);