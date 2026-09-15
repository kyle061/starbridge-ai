const DEFAULT_PUBLIC_VERSION = '0.01'

/**
 * Keep the brand-facing version compact while leaving the full build version
 * available for update checks and rollback operations.
 */
export function formatPublicVersion(version: string): string {
  const match = version.trim().match(/(?:^|\D)(\d+)\.(\d+)/)
  if (!match) return DEFAULT_PUBLIC_VERSION

  const major = String(Number(match[1]))
  const minor = match[2].padStart(2, '0').slice(0, 2)
  return `${major}.${minor}`
}

