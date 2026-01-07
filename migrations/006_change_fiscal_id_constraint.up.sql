-- Drop the existing global UNIQUE constraint on fiscal_id
-- PostgreSQL generates constraint names like 'receipts_fiscal_id_key' for inline UNIQUE constraints
DO $$
DECLARE
    constraint_name text;
BEGIN
    -- Find the constraint name for the UNIQUE constraint on fiscal_id
    SELECT conname INTO constraint_name
    FROM pg_constraint
    WHERE conrelid = 'receipts'::regclass
      AND contype = 'u'
      AND array_length(conkey, 1) = 1
      AND (SELECT attname FROM pg_attribute WHERE attrelid = conrelid AND attnum = conkey[1]) = 'fiscal_id';
    
    -- Drop the constraint if it exists
    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE receipts DROP CONSTRAINT %I', constraint_name);
    END IF;
END $$;

-- Create a composite UNIQUE constraint on (user_id, fiscal_id)
-- This allows different users to have receipts with the same fiscal_id
-- but prevents the same user from having duplicate receipts
ALTER TABLE receipts ADD CONSTRAINT receipts_user_id_fiscal_id_unique UNIQUE (user_id, fiscal_id);

