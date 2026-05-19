const OAUTH_AFFILIATE_STORAGE_KEY = 'sub2api_oauth_aff_code'

function firstString(values: unknown[]): string {
  for (const value of values) {
    if (Array.isArray(value)) {
      const nested = firstString(value)
      if (nested) return nested
      continue
    }
    if (typeof value === 'string') {
      const trimmed = value.trim()
      if (trimmed) return trimmed
    }
  }
  return ''
}

export function resolveAffiliateReferralCode(...values: unknown[]): string {
  return firstString(values)
}

export function storeOAuthAffiliateCode(code: string | null | undefined): void {
  const normalized = (code || '').trim()
  if (!normalized) {
    localStorage.removeItem(OAUTH_AFFILIATE_STORAGE_KEY)
    return
  }
  localStorage.setItem(OAUTH_AFFILIATE_STORAGE_KEY, normalized)
}

export function loadOAuthAffiliateCode(): string {
  return (localStorage.getItem(OAUTH_AFFILIATE_STORAGE_KEY) || '').trim()
}

export function clearAllAffiliateReferralCodes(): void {
  localStorage.removeItem(OAUTH_AFFILIATE_STORAGE_KEY)
}

export function oauthAffiliatePayload(code: string | null | undefined): { aff_code?: string } {
  const normalized = (code || '').trim()
  return normalized ? { aff_code: normalized } : {}
}
