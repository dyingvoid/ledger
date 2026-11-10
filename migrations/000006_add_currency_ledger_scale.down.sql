BEGIN;

ALTER TABLE ledger.currency
    DROP CONSTRAINT currency_asset_scale_valid,
    DROP CONSTRAINT currency_ledger_unique,
    DROP CONSTRAINT currency_ledger_positive,
    DROP COLUMN asset_scale,
    DROP COLUMN ledger;

COMMIT;
