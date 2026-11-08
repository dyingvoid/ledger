BEGIN;

CREATE FUNCTION ledger.add_deposit(
    p_account_id BIGINT,
    p_amount_minor BIGINT,
    p_currency TEXT
) RETURNS TABLE (journal_entry_id BIGINT) AS $$
#variable_conflict use_column
DECLARE
    v_bank_account_id BIGINT;
    v_client_account_type INTEGER;
    v_client_currency TEXT;
    v_journal_entry_id BIGINT;
BEGIN
    IF p_amount_minor <= 0 THEN
        RAISE EXCEPTION 'deposit amount must be positive'
            USING ERRCODE = '22023';
    END IF;

    SELECT a.account_type, a.currency
    INTO v_client_account_type, v_client_currency
    FROM ledger.account a
    WHERE a.id = p_account_id
      AND a.deleted_at IS NULL
    FOR UPDATE;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    IF v_client_account_type <> 2100 THEN
        RAISE EXCEPTION 'account % is not a client deposit account', p_account_id
            USING ERRCODE = '22023';
    END IF;

    IF v_client_currency <> p_currency THEN
        RAISE EXCEPTION 'account % currency % does not match deposit currency %',
            p_account_id, v_client_currency, p_currency
            USING ERRCODE = '22023';
    END IF;

    SELECT a.id
    INTO v_bank_account_id
    FROM ledger.account a
    WHERE a.account_type = 1000
      AND a.currency = p_currency
      AND a.deleted_at IS NULL
    ORDER BY a.id
    LIMIT 1
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'bank cash account for currency % not found', p_currency;
    END IF;

    INSERT INTO ledger.journal_entry (entry_type, entry_date)
    VALUES ('deposit', CURRENT_DATE)
    RETURNING id INTO v_journal_entry_id;

    INSERT INTO ledger.journal_line (journal_entry_id, account_id, side, amount_minor)
    VALUES
        (v_journal_entry_id, v_bank_account_id, 'DR', p_amount_minor),
        (v_journal_entry_id, p_account_id, 'CR', p_amount_minor);

    UPDATE ledger.account
    SET dr = dr + p_amount_minor
    WHERE id = v_bank_account_id;

    UPDATE ledger.account
    SET cr = cr + p_amount_minor
    WHERE id = p_account_id;

    RETURN QUERY SELECT v_journal_entry_id;
END;
$$ LANGUAGE plpgsql;

COMMIT;
