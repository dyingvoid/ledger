BEGIN;

CREATE SCHEMA ledger;

CREATE TABLE ledger.account_type (
    code INTEGER PRIMARY KEY CHECK (code BETWEEN 1000 AND 9999),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE ledger.currency (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT ''
);

INSERT INTO ledger.currency (code, name) VALUES
    ('USD', 'US Dollar'),
    ('CNY', 'Chinese Yuan'),
    ('RUB', 'Russian Ruble');

CREATE TABLE ledger.account (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    currency TEXT NOT NULL REFERENCES ledger.currency (code),
    cr BIGINT NOT NULL DEFAULT 0,
    dr BIGINT NOT NULL DEFAULT 0,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ledger.journal_entry (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    entry_date DATE NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ledger.journal_line (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    journal_entry_id BIGINT NOT NULL REFERENCES ledger.journal_entry (id),
    account_id BIGINT NOT NULL REFERENCES ledger.account (id),
    side TEXT NOT NULL CHECK (side IN ('DR', 'CR')),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0)
);

CREATE INDEX journal_line_journal_entry_id_idx ON ledger.journal_line (journal_entry_id);
CREATE INDEX journal_line_account_id_idx ON ledger.journal_line (account_id);
CREATE INDEX account_active_idx ON ledger.account (id) WHERE deleted_at IS NULL;

CREATE FUNCTION ledger.reject_delete() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION '% cannot be deleted', TG_TABLE_NAME;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER currency_reject_delete
    BEFORE DELETE ON ledger.currency
    FOR EACH ROW EXECUTE FUNCTION ledger.reject_delete();

CREATE TRIGGER account_reject_delete
    BEFORE DELETE ON ledger.account
    FOR EACH ROW EXECUTE FUNCTION ledger.reject_delete();

CREATE TRIGGER journal_entry_reject_delete
    BEFORE DELETE ON ledger.journal_entry
    FOR EACH ROW EXECUTE FUNCTION ledger.reject_delete();

CREATE TRIGGER journal_line_reject_delete
    BEFORE DELETE ON ledger.journal_line
    FOR EACH ROW EXECUTE FUNCTION ledger.reject_delete();

COMMIT;
