-- Reverse email verification additions
DROP INDEX IF EXISTS idx_email_verifications_code_hash;
DROP INDEX IF EXISTS idx_email_verifications_email;
DROP INDEX IF EXISTS idx_email_verifications_user_id;
DROP TABLE IF EXISTS email_verifications;

-- Reverse authorization additions 
DROP INDEX IF EXISTS idx_casbin_rule_ptype;
DROP TABLE IF EXISTS casbin_rule;

-- Finally drop users and their indexes
DROP INDEX IF EXISTS idx_users_google_id_active;
DROP INDEX IF EXISTS idx_users_username_active;
DROP INDEX IF EXISTS idx_users_email_active;
DROP TABLE IF EXISTS users;
