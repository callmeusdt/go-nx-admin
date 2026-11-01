export function siteTitle(): string {
  return document.querySelector('meta[name="nx-site-title"]')?.getAttribute('content')?.trim() ?? ''
}
