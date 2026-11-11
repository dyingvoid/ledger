BEGIN;

INSERT INTO ledger.account_type (code, name, description) VALUES
    (1000, 'Bank''s Cash', 'Asset account for bank cash'),
    (2100, 'Client''s Deposit', 'Liability account for client deposits');

INSERT INTO ledger.currency_info (code, name, ledger, asset_scale) VALUES
    ('USD', 'US Dollar', 1, 2),
    ('CNY', 'Chinese Yuan', 2, 2),
    ('RUB', 'Russian Ruble', 3, 2);

COMMIT;
