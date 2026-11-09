/** Format an API timestamp in the browser timezone unless an IANA timezone is supplied. */
export function formatTimestamp(value: string, locale: string, timeZone?: string): string {
  return new Date(value).toLocaleString(locale, timeZone ? { timeZone } : undefined)
}

/** Format an API timestamp as a date, preserving the existing locale format. */
export function formatTimestampDate(value: string, locale: string, timeZone?: string): string {
  return new Date(value).toLocaleDateString(locale, timeZone ? { timeZone } : undefined)
}
