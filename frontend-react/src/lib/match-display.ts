import type { Match } from '@/lib/api'

export type MatchFilter = 'live' | 'finished' | 'upcoming'

export type MatchLabels = {
  homeTeam: string
  awayTeam: string
  league: string
  kickoff?: string
  isLive: boolean
  scoreLine?: string
}

export type MatchScores = {
  home: number | null
  away: number | null
  hasScore: boolean
}

const LIVE_STATES = new Set([
  'inprogress',
  'in_progress',
  'live',
  'halftime',
  'ht',
  'ht2',
  'htet',
  'extratime',
  'penalties',
  'i',
  'i2',
  'et1',
  'et2',
  'p',
  'pe',
  'h1',
  'h2',
  'h11',
  'h21',
])

const FINAL_STATES = new Set([
  'f',
  'f1',
  'f2',
  'fet',
  'fpe',
  'ft',
  'finished',
  'fulltime',
  'a',
  'a1',
  'a2',
])

const UPCOMING_GRACE_MS = 5 * 60 * 1000
// Align with backend cache.InferFinalState (105 minutes).
const MATCH_DURATION_MS = 105 * 60 * 1000

export function matchScores(match: Match): MatchScores {
  const hasScore =
    match.home_goals != null &&
    match.away_goals != null &&
    !Number.isNaN(match.home_goals) &&
    !Number.isNaN(match.away_goals)

  return {
    home: hasScore ? match.home_goals! : null,
    away: hasScore ? match.away_goals! : null,
    hasScore,
  }
}

export function kickoffMs(match: Match): number {
  return match.start_time ?? 0
}

export function hasKickoffPassed(match: Match, now = Date.now()): boolean {
  const kickoff = kickoffMs(match)
  return kickoff > 0 && kickoff <= now
}

export function isMatchFinished(match: Match, now = Date.now()): boolean {
  if (match.is_final) return true

  const status = match.status.toLowerCase().trim()
  if (FINAL_STATES.has(status)) return true

  const kickoff = kickoffMs(match)
  if (
    kickoff > 0 &&
    kickoff <= now &&
    now - kickoff >= MATCH_DURATION_MS
  ) {
    return true
  }

  return false
}

export function classifyMatch(match: Match, now = Date.now()): MatchFilter {
  if (isMatchFinished(match, now)) return 'finished'

  const status = match.status.toLowerCase().trim()
  const scores = matchScores(match)
  const kickoff = kickoffMs(match)
  const kickoffPassed = kickoff > 0 && kickoff <= now

  if (LIVE_STATES.has(status)) return 'live'
  if (kickoffPassed && (scores.hasScore || match.seq > 0)) return 'live'

  if (kickoff > now + UPCOMING_GRACE_MS) return 'upcoming'
  if (!kickoffPassed && (status === 'scheduled' || status === 'ns' || status === 'ns2')) {
    return 'upcoming'
  }

  return kickoffPassed ? 'live' : 'upcoming'
}

export function filterMatches(matches: Match[], filter: MatchFilter): Match[] {
  return matches.filter((match) => classifyMatch(match) === filter)
}

export function countMatchesByFilter(matches: Match[]): Record<MatchFilter, number> {
  return matches.reduce(
    (acc, match) => {
      acc[classifyMatch(match)] += 1
      return acc
    },
    { live: 0, finished: 0, upcoming: 0 } satisfies Record<MatchFilter, number>,
  )
}

export type CompetitionGroup = {
  competition: string
  matches: Match[]
}

export function groupMatchesByCompetition(matches: Match[]): CompetitionGroup[] {
  const groups = new Map<string, Match[]>()

  for (const match of matches) {
    const key = match.competition?.trim() || 'Fixtures'
    const bucket = groups.get(key)
    if (bucket) bucket.push(match)
    else groups.set(key, [match])
  }

  return Array.from(groups.entries())
    .map(([competition, groupMatches]) => ({
      competition,
      matches: groupMatches.sort((a, b) => kickoffMs(a) - kickoffMs(b)),
    }))
    .sort((a, b) => a.competition.localeCompare(b.competition))
}

export function matchLabels(match: Match): MatchLabels {
  const scores = matchScores(match)

  const homeName = match.home_team?.trim() || 'Home'
  const awayName = match.away_team?.trim() || 'Away'

  const scoreLine = scores.hasScore
    ? `${scores.home} – ${scores.away}`
    : undefined

  const league =
    match.competition?.trim() ||
    (match.fixture_id ? `Fixture ${match.fixture_id}` : 'Fixture')

  const kickoff =
    match.start_time && match.start_time > 0
      ? new Date(match.start_time).toISOString().replace('.000Z', 'Z')
      : undefined

  return {
    homeTeam: homeName,
    awayTeam: awayName,
    league,
    kickoff,
    isLive: classifyMatch(match) === 'live',
    scoreLine,
  }
}

export function formatKickoffDate(match: Match): string {
  if (!match.start_time || match.start_time <= 0) return '—'
  return new Intl.DateTimeFormat(undefined, {
    day: '2-digit',
    month: 'short',
  }).format(new Date(match.start_time))
}

export function formatKickoffClock(match: Match): string {
  if (!match.start_time || match.start_time <= 0) return '—'
  return new Intl.DateTimeFormat(undefined, {
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  }).format(new Date(match.start_time))
}

export function formatStatusShort(match: Match): string {
  if (isMatchFinished(match)) return 'FT'

  const status = match.status.toLowerCase()
  if (!hasKickoffPassed(match) && (status === 'scheduled' || status === 'ns2')) {
    return '—'
  }
  if (status === 'ht' || status === 'ht2') return 'HT'
  if (LIVE_STATES.has(status)) return status.toUpperCase().slice(0, 3)
  if (hasKickoffPassed(match) && classifyMatch(match) === 'live') return 'LIVE'

  return '—'
}

export function formatMatchStatus(match: Match): string {
  if (isMatchFinished(match)) return 'Final'
  if (match.status.toLowerCase() === 'scheduled') return 'Scheduled'
  return match.status.replace(/_/g, ' ')
}

export function formatKickoff(match: Match): string | undefined {
  if (!match.start_time || match.start_time <= 0) return undefined
  return new Intl.DateTimeFormat(undefined, {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
    timeZoneName: 'short',
  }).format(new Date(match.start_time))
}

export function formatOdds(value: number | null | undefined): string {
  if (value == null || Number.isNaN(value)) return '—'
  return value < 10 ? value.toFixed(2) : value.toFixed(1)
}