BEGIN;

INSERT INTO ledger.currency_info (code, name, ledger, asset_scale) VALUES
    ('USD', 'US Dollar', 1, 2),
    ('RUB', 'Russian Ruble', 3, 2),
    ('CNY', 'Chinese Yuan', 2, 2);

COMMIT;
