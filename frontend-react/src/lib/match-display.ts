import type { Match } from '@/lib/api'

export type MatchFilter = 'live' | 'finished' | 'upcoming'

export type MatchLabels = {
  homeTeam: string
  awayTeam: string
  league: string
  kickoff?: string
  isLive: boolean
  isStatusStale: boolean
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
  'aet',
])

const UPCOMING_GRACE_MS = 5 * 60 * 1000
const MAX_LIVE_STATUS_AGE_MS = 4 * 60 * 60 * 1000

const STATUS_SHORT_MAP: Record<string, string> = {
  scheduled: '—',
  ns: '—',
  ns2: '—',
  ht: 'HT',
  ht2: 'HT',
  halftime: 'HT',
  htet: 'AET',
  extratime: 'AET',
  et1: 'AET',
  et2: 'AET',
  penalties: 'PEN',
  pe: 'PEN',
  p: 'PEN',
  inprogress: 'LIVE',
  in_progress: 'LIVE',
  live: 'LIVE',
  i: 'LIVE',
  i2: 'LIVE',
  h1: '1H',
  h2: '2H',
  h11: '1H',
  h21: '2H',
}

const STATUS_LABEL_MAP: Record<string, string> = {
  scheduled: 'Scheduled',
  ns: 'Scheduled',
  ns2: 'Scheduled',
  ht: 'Half-time',
  ht2: 'Half-time',
  halftime: 'Half-time',
  htet: 'Extra-time break',
  extratime: 'After extra time',
  et1: 'Extra time',
  et2: 'Extra time',
  penalties: 'Penalties',
  pe: 'Penalties',
  p: 'Penalties',
  inprogress: 'In progress',
  in_progress: 'In progress',
  live: 'Live',
  i: 'In progress',
  i2: 'In progress',
  h1: 'First half',
  h2: 'Second half',
  h11: 'First half',
  h21: 'Second half',
}

function normalizedStatus(match: Match): string {
  return match.status.toLowerCase().trim()
}

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

export function isMatchFinished(match: Match): boolean {
  if (match.is_final) return true

  const status = normalizedStatus(match)
  if (FINAL_STATES.has(status)) return true

  return false
}

export function isMatchStatusStale(match: Match, now = Date.now()): boolean {
  if (match.is_final) return false
  if (match.status_stale) return true
  const kickoff = kickoffMs(match)
  return kickoff > 0 && now - kickoff >= MAX_LIVE_STATUS_AGE_MS
}

export function classifyMatch(match: Match, now = Date.now()): MatchFilter {
  if (isMatchFinished(match)) return 'finished'
  if (isMatchStatusStale(match, now)) return 'finished'

  const status = normalizedStatus(match)
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
    isStatusStale: isMatchStatusStale(match),
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
  const status = normalizedStatus(match)
  if (isMatchStatusStale(match)) return 'PEND'
  if (isMatchFinished(match)) {
    if (status === 'extratime' || status === 'fet' || status === 'aet') return 'AET'
    if (status === 'fpe' || status === 'penalties') return 'PEN'
    return 'FT'
  }

  if ((status === 'scheduled' || status === 'ns' || status === 'ns2') && hasKickoffPassed(match)) {
    return classifyMatch(match) === 'live' ? 'LIVE' : '—'
  }
  const mapped = STATUS_SHORT_MAP[status]
  if (mapped) return mapped
  if (hasKickoffPassed(match) && classifyMatch(match) === 'live') return 'LIVE'

  return '—'
}

export function formatMatchStatus(match: Match): string {
  const status = normalizedStatus(match)
  if (isMatchStatusStale(match)) return 'Result pending'
  if (isMatchFinished(match)) {
    if (status === 'extratime' || status === 'fet' || status === 'aet') return 'Final AET'
    if (status === 'fpe' || status === 'penalties') return 'Final PEN'
    return 'Final'
  }

  if ((status === 'scheduled' || status === 'ns' || status === 'ns2') && hasKickoffPassed(match)) {
    return classifyMatch(match) === 'live' ? 'Live' : 'Scheduled'
  }
  const mapped = STATUS_LABEL_MAP[status]
  if (mapped) return mapped
  if (status === 'scheduled') return 'Scheduled'
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
