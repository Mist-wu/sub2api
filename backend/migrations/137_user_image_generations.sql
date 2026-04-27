CREATE TABLE IF NOT EXISTS user_image_generations (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  prompt TEXT NOT NULL,
  revised_prompt TEXT,
  model VARCHAR(100) NOT NULL DEFAULT 'gpt-image-2',
  mime_type VARCHAR(100) NOT NULL DEFAULT 'image/png',
  image_data BYTEA NOT NULL,
  image_sha256 VARCHAR(64) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_image_generations_user_id ON user_image_generations(user_id);
CREATE INDEX IF NOT EXISTS idx_user_image_generations_created_at ON user_image_generations(created_at);
CREATE INDEX IF NOT EXISTS idx_user_image_generations_user_created ON user_image_generations(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_image_generations_image_sha256 ON user_image_generations(image_sha256);
