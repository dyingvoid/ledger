BEGIN;

DELETE FROM ledger.account_type WHERE code IN (1000, 2100);

COMMIT;
