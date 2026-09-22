import { getStaticTOMLValue, parseTOML, type AST } from 'toml-eslint-parser'

export type CodexConnectionErrorCode = 'invalid' | 'provider' | 'auth' | 'profile'

export class CodexConnectionError extends Error {
  constructor(public code: CodexConnectionErrorCode) {
    super(code)
  }
}

// Edit value ranges instead of serializing TOML: comments, custom settings and
// the existing provider identity must survive a connection/key change verbatim.
export function updateCodexConnection(source: string, endpoint: string, apiKey: string): string {
  let ast: AST.TOMLProgram
  try {
    ast = parseTOML(source, { tomlVersion: '1.0' })
  } catch {
    throw new CodexConnectionError('invalid')
  }
  const entries: { path: string[]; node: AST.TOMLKeyValue }[] = []
  const visit = (node: AST.TOMLKeyValue, parent: string[]) => {
    const path = [...parent, ...getStaticTOMLValue(node.key)]
    entries.push({ path, node })
    if (node.value.type === 'TOMLInlineTable') {
      node.value.body.forEach(child => visit(child, path))
    }
  }
  const tables: AST.TOMLTable[] = []
  for (const node of ast.body[0].body) {
    if (node.type === 'TOMLTable') {
      tables.push(node)
      node.body.forEach(child => visit(child, node.resolvedKey.map(String)))
    } else visit(node, [])
  }
  const equal = (a: string[], b: string[]) => a.length === b.length && a.every((key, i) => key === b[i])
  const find = (path: string[]) => entries.find(entry => equal(entry.path, path))?.node
  const stringValue = (node?: AST.TOMLKeyValue) => node?.value.type === 'TOMLValue' && node.value.kind === 'string' ? node.value.value : undefined
  const provider = stringValue(find(['model_provider']))
  if (!provider || ['openai', 'ollama', 'lmstudio'].includes(provider)) throw new CodexConnectionError('provider')
  const path = ['model_providers', provider]
  const table = tables.find(node => equal(node.resolvedKey.map(String), path) && node.kind === 'standard')
  const inline = find(path)?.value
  const providerExists = table || inline?.type === 'TOMLInlineTable' || entries.some(entry => equal(entry.path.slice(0, 2), path))
  if (!providerExists) throw new CodexConnectionError('provider')
  // A profile can override the connection selected by the root settings. Do not
  // silently modify a different profile/provider or change profile selection.
  const profile = stringValue(find(['profile']))
  if (profile && entries.some(entry => entry.path[0] === 'profiles' && entry.path[1] === profile && ['model_provider', 'model_providers'].includes(entry.path[2]))) {
    throw new CodexConnectionError('profile')
  }
  const usesOtherAuth = (keys: string[]) => equal(keys.slice(0, 2), path) && (
    ['env_key', 'auth', 'aws'].includes(keys[2]) ||
    (['http_headers', 'env_http_headers'].includes(keys[2]) && ['authorization', 'x-api-key', 'api-key'].includes(keys[3]?.toLowerCase()))
  )
  if (entries.some(entry => usesOtherAuth(entry.path)) || tables.some(table => usesOtherAuth(table.resolvedKey.map(String)))) {
    throw new CodexConnectionError('auth')
  }
  const newline = source.includes('\r\n') ? '\r\n' : '\n'
  const quote = (value: string) => JSON.stringify(value).replace(/\u007f/g, '\\u007f')
  const replacements: { start: number; end: number; value: string }[] = []
  const missing: string[] = []
  for (const [key, value] of [['base_url', endpoint], ['experimental_bearer_token', apiKey]]) {
    const node = find([...path, key])
    if (node) {
      if (stringValue(node) === undefined) throw new CodexConnectionError('invalid')
      replacements.push({ start: node.value.range[0], end: node.value.range[1], value: quote(value) })
    } else missing.push(`${key} = ${quote(value)}`)
  }
  if (missing.length) {
    if (inline?.type === 'TOMLInlineTable') {
      replacements.push({ start: inline.range[1] - 1, end: inline.range[1] - 1, value: `${inline.body.length ? ', ' : ''}${missing.join(', ')} ` })
    } else if (table) {
      const endOfHeader = source.indexOf('\n', table.key.range[1])
      const start = endOfHeader === -1 ? source.length : endOfHeader + 1
      replacements.push({ start, end: start, value: `${endOfHeader === -1 ? newline : ''}${missing.join(newline)}${newline}` })
    } else {
      // Dotted-key configurations can only be extended from the root. Inline
      // ancestor tables are closed and need manual editing instead.
      if (entries.some(entry => entry.node.value.type === 'TOMLInlineTable' && equal(path.slice(0, entry.path.length), entry.path))) throw new CodexConnectionError('provider')
      const start = tables[0]?.range[0] ?? source.length
      const prefix = path.map(quote).join('.')
      replacements.push({ start, end: start, value: `${start > 0 && source[start - 1] !== '\n' ? newline : ''}${missing.map(line => `${prefix}.${line}`).join(newline)}${newline}` })
    }
  }
  let updated = source
  for (const edit of replacements.sort((a, b) => b.start - a.start)) updated = updated.slice(0, edit.start) + edit.value + updated.slice(edit.end)
  try { parseTOML(updated, { tomlVersion: '1.0' }) } catch { throw new CodexConnectionError('invalid') }
  return updated
}
