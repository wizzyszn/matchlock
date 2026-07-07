import type { Match, Side } from '@/lib/api'
import { matchLabels } from '@/lib/match-display'

export const ALL_SIDES: Side[] = ['home', 'draw', 'away']

export function sideLabel(side: Side, match: Match): string {
  const labels = matchLabels(match)
  switch (side) {
    case 'home':
      return labels.homeTeam
    case 'away':
      return labels.awayTeam
    case 'draw':
      return 'Draw'
  }
}

export function sideShortLabel(side: Side): string {
  switch (side) {
    case 'home':
      return '1'
    case 'draw':
      return 'X'
    case 'away':
      return '2'
  }
}

export function availableTakerSides(makerSide: Side): Side[] {
  return ALL_SIDES.filter((side) => side !== makerSide)
}

export function defaultTakerSide(makerSide: Side): Side {
  const options = availableTakerSides(makerSide)
  return options[0] ?? 'away'
}

export function referenceOddsForSide(
  side: Side,
  odds?: Match['odds'],
): number | null {
  if (!odds) return null
  switch (side) {
    case 'home':
      return odds.home ?? null
    case 'draw':
      return odds.draw ?? null
    case 'away':
      return odds.away ?? null
  }
}