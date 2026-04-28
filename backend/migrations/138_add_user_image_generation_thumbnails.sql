ALTER TABLE user_image_generations
    ADD COLUMN IF NOT EXISTS thumbnail_data BYTEA,
    ADD COLUMN IF NOT EXISTS thumbnail_mime_type VARCHAR(100);
