-- +migrate Up
DROP TABLE IF EXISTS purchase_receipt;

CREATE TABLE IF NOT EXISTS purchase (
    PRIMARY KEY (transaction_id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL,

    create_time    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    environment    SMALLINT     NOT NULL DEFAULT 0, -- Unknown(0), Sandbox(1), Production(2)
    product_id     VARCHAR(512) NOT NULL,
    purchase_time  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    raw_response   JSONB        NOT NULL DEFAULT '{}',
    store          SMALLINT     NOT NULL DEFAULT 0, -- AppleAppStore(0), GooglePlay(1), Huawei(2)
    transaction_id VARCHAR(512) NOT NULL CHECK (length(transaction_id) > 0),
    update_time    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    user_id        UUID         DEFAULT NULL
);
CREATE INDEX IF NOT EXISTS purchase_user_id_purchase_time_transaction_id_idx
    ON purchase (user_id, purchase_time DESC, transaction_id);

-- +migrate Down
DROP TABLE IF EXISTS purchase;

CREATE TABLE IF NOT EXISTS purchase_receipt (
    PRIMARY KEY (receipt),

    create_time    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    product_id     VARCHAR(512) NOT NULL,
    purchase_time  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    raw_response   JSONB        NOT NULL DEFAULT '{}',
    receipt        TEXT         NOT NULL CHECK (length(receipt) > 0),
    store          SMALLINT     NOT NULL DEFAULT 0, -- AppleAppStore(0), GooglePlay(1), Huawei(2)
    transaction_id VARCHAR(512) NOT NULL CHECK (length(transaction_id) > 0),
    update_time    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    user_id        UUID         NOT NULL
);
CREATE INDEX IF NOT EXISTS purchase_receipt_user_id_purchase_time_transaction_id_idx
    ON purchase_receipt (user_id, purchase_time DESC, transaction_id);
