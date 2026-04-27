-- Add GPT-5.5 and GPT Image 2 to existing OpenAI OAuth model mappings.
--
-- New OpenAI accounts already get these models from the built-in default catalog
-- when no model_mapping is configured. Existing OAuth/setup-token accounts that
-- use model_mapping as an allowlist need the aliases added explicitly.

UPDATE accounts
SET credentials = jsonb_set(
    credentials,
    '{model_mapping}',
    credentials->'model_mapping'
      || CASE
        WHEN credentials->'model_mapping' ? 'gpt-5.5' THEN '{}'::jsonb
        ELSE '{"gpt-5.5": "gpt-5.5"}'::jsonb
      END
      || CASE
        WHEN credentials->'model_mapping' ? 'gpt-image-2' THEN '{}'::jsonb
        ELSE '{"gpt-image-2": "gpt-image-2"}'::jsonb
      END
)
WHERE platform = 'openai'
  AND type IN ('oauth', 'setup-token')
  AND deleted_at IS NULL
  AND jsonb_typeof(credentials->'model_mapping') = 'object'
  AND (
    NOT (credentials->'model_mapping' ? 'gpt-5.5')
    OR NOT (credentials->'model_mapping' ? 'gpt-image-2')
  );
