BEGIN;

INSERT INTO ledger.account (user_id, name, description, account_type, currency)
SELECT
    228,
    'deposit-seed-' || g,
    'loadtest deposit seed (migration 000005)',
    2100,
    'USD'
FROM generate_series(1, 15000000) AS g;

COMMIT;
