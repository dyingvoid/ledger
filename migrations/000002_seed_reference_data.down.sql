BEGIN;

DELETE FROM ledger.currency_info WHERE code IN ('USD', 'CNY', 'RUB');

DELETE FROM ledger.account_type WHERE code IN (1000, 2100);

COMMIT;
