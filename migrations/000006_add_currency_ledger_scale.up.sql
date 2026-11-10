BEGIN;

ALTER TABLE ledger.currency
    ADD COLUMN ledger INTEGER,
    ADD COLUMN asset_scale SMALLINT;

UPDATE ledger.currency
SET ledger = CASE code
        WHEN 'USD' THEN 1
        WHEN 'CNY' THEN 2
        WHEN 'RUB' THEN 3
    END,
    asset_scale = 2
WHERE code IN ('USD', 'CNY', 'RUB');

ALTER TABLE ledger.currency
    ALTER COLUMN ledger SET NOT NULL,
    ALTER COLUMN asset_scale SET NOT NULL,
    ADD CONSTRAINT currency_ledger_positive CHECK (ledger > 0),
    ADD CONSTRAINT currency_ledger_unique UNIQUE (ledger),
    ADD CONSTRAINT currency_asset_scale_valid CHECK (asset_scale BETWEEN 0 AND 18);

COMMIT;
