-- Remove the seeded DeepSeek route that existed only for the GPT-6 requirements pass.
-- Preserve custom routes, other public models, and historical usage records.
UPDATE composite_model_routes AS route
SET enabled = false,
    deleted_at = NOW(),
    updated_at = NOW()
FROM groups AS relay
WHERE route.group_id = relay.id
  AND relay.name = 'composite-default'
  AND relay.platform = 'composite'
  AND relay.description = 'Default OpenAI + DeepSeek relay'
  AND relay.deleted_at IS NULL
  AND route.deleted_at IS NULL
  AND route.enabled = true
  AND route.public_model IN ('gpt-6', 'gpt-6-astra')
  AND route.match_type = 'exact'
  AND route.endpoint = 'any'
  AND route.target_platform = 'deepseek'
  AND route.upstream_model = 'deepseek-v4-pro'
  AND route.priority = 20
  AND route.notes = 'DeepSeek requirements preparation fallback';
