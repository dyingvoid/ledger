BEGIN;

INSERT INTO ledger.account_type (code, name, description) VALUES
    (1000, 'Bank''s Cash', 'Asset account for bank cash'),
    (2100, 'Client''s Deposit', 'Liability account for client deposits');

INSERT INTO ledger.entry_type (code, name, description) VALUES
    ('deposit', 'Deposit', 'Money deposited into an account'),
    ('transfer', 'Transfer', 'Money moved between accounts'),
    ('withdrawal', 'Withdrawal', 'Money withdrawn from an account');

INSERT INTO ledger.currency (code, name) VALUES
    ('USD', 'US Dollar'),
    ('CNY', 'Chinese Yuan'),
    ('RUB', 'Russian Ruble');

INSERT INTO ledger.account (user_id, name, description, account_type, currency)
SELECT 0, c.name || ' Cash', 'Bank cash account for ' || c.name, 1000, c.code
FROM ledger.currency c;

COMMIT;
