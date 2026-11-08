BEGIN;

INSERT INTO ledger.account_type (code, name, description) VALUES
    (1000, 'Bank''s Cash', 'Asset account for bank cash'),
    (2100, 'Client''s Deposit', 'Liability account for client deposits');

COMMIT;
