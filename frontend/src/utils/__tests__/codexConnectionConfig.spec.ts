import { describe, expect, it } from 'vitest'
import { getStaticTOMLValue, parseTOML } from 'toml-eslint-parser'
import { CodexConnectionError, updateCodexConnection } from '../codexConnectionConfig'

const endpoint = 'https://starbridge.example/v1'
const key = 'sk-new-test'
const original = `# My existing setup
model = "gpt-6-astra"
model_provider = "zero-inference" # preserve session identity
model_reasoning_effort = "xhigh"
personality = "pragmatic"
service_tier = "default"

[model_providers.zero-inference] # keep provider
name = "My Provider"
base_url = 'https://old.example/v1' # endpoint
experimental_bearer_token = 'sk-old-test' # key
wire_api = "responses"
supports_websockets = true
requires_openai_auth = true

[features]
responses_websockets_v2 = true
goals = true
[mcp_servers.local]
command = "example"
args = ["keep", "all"]
[projects."/Users/example/My Project"]
trust_level = "trusted"
[history]
persistence = "save-all"
`

describe('updateCodexConnection', () => {
  it('changes only endpoint and token value ranges, preserving all other bytes', () => {
    expect(updateCodexConnection(original, endpoint, key)).toBe(original
      .replace("'https://old.example/v1'", JSON.stringify(endpoint))
      .replace("'sk-old-test'", JSON.stringify(key)))
  })

  it('preserves CRLF, quoted provider IDs, comments, multiline content and other providers', () => {
    const source = `model_provider = 'old.name'\r\n[model_providers."old.name"] # header\r\nbase_url = "old"\r\nexperimental_bearer_token = "old-key"\r\nsupports_websockets = false\r\nrequires_openai_auth = true\r\n[model_providers.other]\r\nbase_url = "unchanged"\r\n[extra]\r\ntext = '''\r\n[model_providers.fake]\r\nbase_url = "fake"\r\n'''\r\n`
    expect(updateCodexConnection(source, endpoint, key)).toBe(source.replace('"old"', JSON.stringify(endpoint)).replace('"old-key"', JSON.stringify(key)))
  })

  it('adds missing connection fields inside the active table before the next section', () => {
    const source = 'model_provider="saved"\n[model_providers.saved] # keep\nname="saved"\n[features]\ngoals=true\n'
    const updated = updateCodexConnection(source, endpoint, key)
    expect(updated).toBe(source.replace('# keep\n', `# keep\nbase_url = "${endpoint}"\nexperimental_bearer_token = "${key}"\n`))
    const parsed = getStaticTOMLValue(parseTOML(updated))
    expect(parsed.features).toEqual({ goals: true })
  })

  it.each([
    'model_provider="saved"\nmodel_providers.saved.base_url="old"\n',
    'model_provider="saved"\n[model_providers]\nsaved={base_url="old",supports_websockets=true,requires_openai_auth=true}\n',
    'model_provider="saved"\n[model_providers.saved]',
  ])('handles dotted keys, inline tables and a header at EOF', source => {
    const updated = updateCodexConnection(source, endpoint, key)
    const parsed = getStaticTOMLValue(parseTOML(updated)) as any
    expect(parsed.model_provider).toBe('saved')
    expect(parsed.model_providers.saved.base_url).toBe(endpoint)
    expect(parsed.model_providers.saved.experimental_bearer_token).toBe(key)
  })

  it('escapes strings without injecting TOML and preserves unicode', () => {
    const token = 'sk-"\\\n中文\t\u0001'
    const parsed = getStaticTOMLValue(parseTOML(updateCodexConnection(original, endpoint, token))) as any
    expect(parsed.model_providers['zero-inference'].experimental_bearer_token).toBe(token)
  })

  it('preserves non-authentication headers and unrelated profiles', () => {
    const source = original + '[model_providers.zero-inference.http_headers]\nX-Custom = "keep"\n[profiles.review]\nmodel = "my-model"\n'
    expect(updateCodexConnection(source, endpoint, key)).toBe(source
      .replace("'https://old.example/v1'", JSON.stringify(endpoint))
      .replace("'sk-old-test'", JSON.stringify(key)))
  })

  it.each([
    ['broken = [', 'invalid'],
    ['model_provider="openai"', 'provider'],
    ['model_provider="missing"', 'provider'],
    [original.replace('wire_api = "responses"', 'env_key = "EXISTING_KEY"'), 'auth'],
    [original.replace('wire_api = "responses"', 'auth = { command = "get-token" }'), 'auth'],
    [original + '[model_providers.zero-inference.aws]\n', 'auth'],
    [original.replace('wire_api = "responses"', 'http_headers = { Authorization = "Bearer old" }'), 'auth'],
    ['profile="work"\n' + original + '[profiles.work]\nmodel_provider="other"\n', 'profile'],
    [original.replace("'https://old.example/v1'", '42'), 'invalid'],
  ])('refuses an unsafe merge without inventing a replacement config', (source, code) => {
    try {
      updateCodexConnection(source, endpoint, key)
      expect.fail('Expected the unsafe merge to be rejected')
    } catch (error) {
      expect(error).toBeInstanceOf(CodexConnectionError)
      expect((error as CodexConnectionError).code).toBe(code)
    }
  })
})
