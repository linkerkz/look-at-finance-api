-- Drop the composite UNIQUE constraint
ALTER TABLE receipts DROP CONSTRAINT IF EXISTS receipts_user_id_fiscal_id_unique;

-- Restore the original global UNIQUE constraint on fiscal_id
ALTER TABLE receipts ADD CONSTRAINT receipts_fiscal_id_key UNIQUE (fiscal_id);

