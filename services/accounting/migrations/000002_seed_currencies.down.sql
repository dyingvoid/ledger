BEGIN;

DELETE FROM ledger.currency_info
WHERE code IN ('USD', 'RUB', 'CNY');

COMMIT;
