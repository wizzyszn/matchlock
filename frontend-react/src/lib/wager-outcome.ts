import type { Match, SettlementState, Side, Wager, WagerSettlementStatus } from '@/lib/api'

export function winningSideFromMatch(match: Match): Side | null {
  if (!match.is_final) return null
  const home = match.home_goals
  const away = match.away_goals
  if (home == null || away == null) return null
  if (home > away) return 'home'
  if (away > home) return 'away'
  if (home === away) return 'draw'
  return null
}

export function userBackedSide(wager: Wager, walletAddress: string): Side {
  if (wager.maker === walletAddress) return wager.maker_side
  return wager.taker_side ?? wager.maker_side
}

export function isWinner(
  wager: Wager,
  match: Match | undefined,
  walletAddress: string,
): boolean {
  if (!match || wager.status !== 'matched') return false
  const outcome = winningSideFromMatch(match)
  if (!outcome) return false
  return userBackedSide(wager, walletAddress) === outcome
}

export function isSettlementClaimable(state: SettlementState | undefined): boolean {
  return (
    state === 'claimable' ||
    state === 'queued' ||
    state === 'retrying' ||
    state === 'failed'
  )
}

export function canClaimWinnings(
  wager: Wager,
  match: Match | undefined,
  walletAddress: string,
  settlement?: WagerSettlementStatus,
): boolean {
  return isWinner(wager, match, walletAddress) && isSettlementClaimable(settlement?.state)
}
