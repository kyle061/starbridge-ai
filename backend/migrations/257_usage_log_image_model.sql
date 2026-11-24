ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS image_model VARCHAR(200);

COMMENT ON COLUMN usage_logs.image_model IS
    'Selected image-generation model, separate from the outer Responses model; NULL for historical or unobserved values';
