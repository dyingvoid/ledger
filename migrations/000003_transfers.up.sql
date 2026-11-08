BEGIN;

CREATE TABLE ledger.transfer (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    journal_entry_id BIGINT NOT NULL REFERENCES ledger.journal_entry (id),
    dr_account_id BIGINT NOT NULL REFERENCES ledger.account (id),
    cr_account_id BIGINT NOT NULL REFERENCES ledger.account (id),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency TEXT NOT NULL REFERENCES ledger.currency (code),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (dr_account_id <> cr_account_id)
);

CREATE INDEX transfer_journal_entry_id_idx ON ledger.transfer (journal_entry_id);
CREATE INDEX transfer_dr_account_id_idx ON ledger.transfer (dr_account_id);
CREATE INDEX transfer_cr_account_id_idx ON ledger.transfer (cr_account_id);

CREATE TRIGGER transfer_reject_delete
    BEFORE DELETE ON ledger.transfer
    FOR EACH ROW EXECUTE FUNCTION ledger.reject_delete();

COMMIT;
