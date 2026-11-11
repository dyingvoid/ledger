BEGIN;

CREATE SCHEMA ledger;

CREATE TABLE ledger.account_type (
    code INTEGER PRIMARY KEY CHECK (code BETWEEN 1000 AND 5999),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE ledger.currency_info (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    ledger INTEGER NOT NULL UNIQUE CHECK (ledger > 0),
    asset_scale SMALLINT NOT NULL CHECK (asset_scale BETWEEN 0 AND 18)
);

CREATE TABLE ledger.account_info (
    id BYTEA PRIMARY KEY CHECK (octet_length(id) = 16),
    user_id BIGINT NOT NULL CHECK (user_id > 0),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    account_type INTEGER NOT NULL REFERENCES ledger.account_type (code),
    currency TEXT NOT NULL REFERENCES ledger.currency_info (code),
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX account_info_user_id_idx ON ledger.account_info (user_id);
CREATE INDEX account_info_currency_idx ON ledger.account_info (currency);
CREATE INDEX account_info_account_type_idx ON ledger.account_info (account_type);

COMMIT;
