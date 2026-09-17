-- Raise the default capacity for the GPT6 two-stage request flow.
-- Preserve explicit low-capacity overrides outside the legacy defaults.
ALTER TABLE users
    ALTER COLUMN concurrency SET DEFAULT 8;

ALTER TABLE accounts
    ALTER COLUMN concurrency SET DEFAULT 10;

UPDATE users
SET concurrency = 8,
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND concurrency = 5;

UPDATE accounts
SET concurrency = 10,
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND concurrency IN (3, 5);

UPDATE settings
SET value = '8'
WHERE key IN (
    'default_concurrency',
    'auth_source_default_email_concurrency',
    'auth_source_default_linuxdo_concurrency',
    'auth_source_default_oidc_concurrency',
    'auth_source_default_wechat_concurrency',
    'auth_source_default_github_concurrency',
    'auth_source_default_google_concurrency',
    'auth_source_default_dingtalk_concurrency'
  )
  AND value = '5';
