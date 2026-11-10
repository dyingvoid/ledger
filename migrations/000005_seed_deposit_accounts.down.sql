BEGIN;

ALTER TABLE ledger.account DISABLE TRIGGER account_reject_delete;

DELETE FROM ledger.account
WHERE user_id = 228
  AND account_type = 2100
  AND currency = 'USD'
  AND description = 'loadtest deposit seed (migration 000005)';

ALTER TABLE ledger.account ENABLE TRIGGER account_reject_delete;

COMMIT;
