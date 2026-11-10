BEGIN;

DELETE FROM ledger.entry_type WHERE code IN ('deposit', 'transfer', 'withdrawal');

DELETE FROM ledger.account_type WHERE code IN (1000, 2100);

COMMIT;
