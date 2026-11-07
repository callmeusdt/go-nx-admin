/** Normalize legacy upload paths and media URLs for previews. */
export function normalizeMediaUrl(value: string | null | undefined): string | undefined {
  if (!value) return undefined
  const trimmed = value.trim()
  if (!trimmed) return undefined
  if (/^https?:\/\//i.test(trimmed)) return trimmed
  if (/^\/?uploads\//i.test(trimmed)) return trimmed.startsWith('/') ? trimmed : `/${trimmed}`
  return undefined
}
