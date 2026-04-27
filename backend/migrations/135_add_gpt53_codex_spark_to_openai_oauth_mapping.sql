-- Add GPT-5.3 Codex Spark to existing OpenAI OAuth model mappings.
--
-- New OpenAI accounts already get this model from the built-in default catalog
-- when no model_mapping is configured. Existing OAuth/setup-token accounts that
-- use model_mapping as an allowlist need the alias added explicitly.

UPDATE accounts
SET credentials = jsonb_set(
    credentials,
    '{model_mapping}',
    credentials->'model_mapping'
      || CASE
        WHEN credentials->'model_mapping' ? 'gpt-5.3-codex-spark' THEN '{}'::jsonb
        ELSE '{"gpt-5.3-codex-spark": "gpt-5.3-codex-spark"}'::jsonb
      END
)
WHERE platform = 'openai'
  AND type IN ('oauth', 'setup-token')
  AND deleted_at IS NULL
  AND jsonb_typeof(credentials->'model_mapping') = 'object'
  AND NOT (credentials->'model_mapping' ? 'gpt-5.3-codex-spark');
