-- Drop trigger
DROP TRIGGER IF EXISTS update_invites_updated_at;

-- Drop indexes
DROP INDEX IF EXISTS idx_invites_expires_at;
DROP INDEX IF EXISTS idx_invites_status;
DROP INDEX IF EXISTS idx_invites_email;
DROP INDEX IF EXISTS idx_invites_family_id;
DROP INDEX IF EXISTS idx_invites_token;

-- Drop invites table
DROP TABLE IF EXISTS invites;
