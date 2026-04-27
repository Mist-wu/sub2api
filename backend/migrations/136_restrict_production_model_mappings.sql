-- Restrict production-facing account model mappings to the curated request allowlist.
-- Empty model_mapping means "all default models", so we write explicit mappings here.

UPDATE accounts
SET
    credentials = jsonb_set(
        COALESCE(credentials, '{}'::jsonb),
        '{model_mapping}',
        '{
          "gpt-5.3-codex": "gpt-5.3-codex",
          "gpt-5.3-codex-spark": "gpt-5.3-codex-spark",
          "gpt-5.4": "gpt-5.4",
          "gpt-5.4-mini": "gpt-5.4-mini",
          "gpt-5.5": "gpt-5.5",
          "gpt-image-2": "gpt-image-2"
        }'::jsonb,
        true
    ),
    updated_at = NOW()
WHERE platform = 'openai'
  AND deleted_at IS NULL;

UPDATE accounts
SET
    credentials = jsonb_set(
        COALESCE(credentials, '{}'::jsonb),
        '{model_mapping}',
        '{
          "gemini-3-flash-preview": "gemini-3-flash-preview",
          "gemini-3-pro-preview": "gemini-3-pro-preview",
          "gemini-3.1-pro-preview": "gemini-3.1-pro-preview"
        }'::jsonb,
        true
    ),
    updated_at = NOW()
WHERE platform = 'gemini'
  AND deleted_at IS NULL;

UPDATE accounts
SET
    credentials = jsonb_set(
        COALESCE(credentials, '{}'::jsonb),
        '{model_mapping}',
        '{
          "gemini-3-flash-preview": "gemini-3-flash",
          "gemini-3-pro-preview": "gemini-3-pro-high",
          "gemini-3.1-pro-preview": "gemini-3.1-pro-high"
        }'::jsonb,
        true
    ),
    updated_at = NOW()
WHERE platform = 'antigravity'
  AND deleted_at IS NULL;
