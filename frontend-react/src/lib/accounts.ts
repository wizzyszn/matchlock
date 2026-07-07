import { PublicKey } from '@solana/web3.js'

export const SYSTEM_PROGRAM_ID = new PublicKey(
  '11111111111111111111111111111111',
)

/** True when a wager has no real matched opponent yet. */
export function isPlaceholderAddress(address: string | undefined | null): boolean {
  if (!address) return true
  try {
    const key = new PublicKey(address)
    if (key.equals(SYSTEM_PROGRAM_ID)) return true
    return key.toBytes().every((byte) => byte === 0)
  } catch {
    return true
  }
}