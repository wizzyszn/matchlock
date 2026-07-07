const DISPLAY_NAME_RE = /^[a-zA-Z0-9_]{3,32}$/

export function isValidDisplayName(value: string): boolean {
  return DISPLAY_NAME_RE.test(value.trim())
}

export function displayNameHint(): string {
  return '3–32 characters: letters, numbers, underscore'
}

export function userDisplayLabel(profile: {
  display_name?: string
  email: string
}): string {
  const name = profile.display_name?.trim()
  if (name) return name
  return profile.email.split('@')[0] ?? profile.email
}

export function needsUsername(profile: {
  display_name?: string
}): boolean {
  return !profile.display_name?.trim()
}

export function truncatePubkey(pubkey: string): string {
  if (pubkey.length <= 10) return pubkey
  return `${pubkey.slice(0, 4)}…${pubkey.slice(-4)}`
}