-- Correct the onboarding GPT6 chain: OpenAI performs the requested task;
-- DeepSeek is reserved for the handler's requirements-preparation stage.
-- Upgrade only unchanged onboarding routes. Preserve operator priorities,
-- disabled routes, custom groups, and historical usage/model records.
UPDATE composite_model_routes AS route
SET priority = 10,
    notes = 'Primary OpenAI GPT6 execution route',
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
  AND route.target_platform = 'openai'
  AND route.upstream_model = 'gpt-6-astra'
  AND route.priority = 20
  AND route.notes IN ('Primary OpenAI Astra route', 'OpenAI fallback when DeepSeek is unavailable');

UPDATE composite_model_routes AS route
SET priority = 20,
    notes = 'DeepSeek requirements preparation fallback',
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
  AND route.priority = 10
  AND route.notes IN ('Primary DeepSeek V4 route for public GPT6 model', 'Primary DeepSeek route');
