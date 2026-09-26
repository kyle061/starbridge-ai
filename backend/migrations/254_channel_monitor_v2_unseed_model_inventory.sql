-- Migration 197 seeded a snapshot of popular model names. It is not an
-- availability catalog and eventually leaves retired models in the settings
-- UI. Return unchanged factory lists to traffic-discovered model names.
-- Compare each platform's sorted seed fingerprint so operator-edited lists,
-- enabled flags and extra platforms are preserved even after other edits.
WITH factory_model_fingerprints(platform, fingerprint) AS (
    VALUES
        ('anthropic', '75941f8a81399f359ac813faf8bbb701'),
        ('openai', '72ee1dbc6ff0289feb8f253a0399c0fe'),
        ('grok', '0aaed6726c7ccdd01af88723872d3324'),
        ('gemini', '43270d7e4652b429c6eff9ef4a296228'),
        ('kiro', '457c94d623eaae07991b9e6675d5975b'),
        ('antigravity', 'ec9411f97aa698caf7d963dfb3d9a207')
), cleaned AS (
    SELECT cfg.id,
           jsonb_agg(
               CASE WHEN factory.fingerprint = md5(COALESCE((
                   SELECT string_agg(model, chr(31) ORDER BY model COLLATE "C")
                   FROM jsonb_array_elements_text(COALESCE(entry.value->'models', '[]'::jsonb)) AS names(model)
               ), ''))
               THEN jsonb_set(entry.value, '{models}', '[]'::jsonb)
               ELSE entry.value END
               ORDER BY entry.ordinality
           ) AS platforms
    FROM channel_monitor_v2_config AS cfg
    CROSS JOIN LATERAL jsonb_array_elements(cfg.platforms) WITH ORDINALITY AS entry(value, ordinality)
    LEFT JOIN factory_model_fingerprints AS factory ON factory.platform = entry.value->>'platform'
    WHERE cfg.id = 1
    GROUP BY cfg.id
)
UPDATE channel_monitor_v2_config AS cfg
SET platforms = cleaned.platforms,
    version = version + 1,
    updated_at = NOW()
FROM cleaned
WHERE cfg.id = cleaned.id
  AND cfg.platforms IS DISTINCT FROM cleaned.platforms;
