import { Swords, Trophy } from 'lucide-react'
import { memo, useEffect, useMemo, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { ChallengeSlipDialog } from '@/components/wager/challenge-slip'
import { TeamRow } from '@/components/wager/team-row'
import { useMediaQuery } from '@/hooks/use-media-query'
import { useMatchesQuery } from '@/hooks/queries/use-matches'
import type { Match, Side } from '@/lib/api'
import {
  classifyMatch,
  countMatchesByFilter,
  filterMatches,
  formatKickoffClock,
  formatKickoffDate,
  formatOdds,
  formatStatusShort,
  groupMatchesByCompetition,
  matchLabels,
  matchScores,
  type MatchFilter,
} from '@/lib/match-display'
import { cn } from '@/lib/utils'

const FILTERS: { id: MatchFilter; label: string }[] = [
  { id: 'live', label: 'Live' },
  { id: 'finished', label: 'Finished' },
  { id: 'upcoming', label: 'Upcoming' },
]

function OddsCell({
  label,
  value,
  highlighted,
  disabled,
  onSelect,
}: {
  label: string
  value: string
  highlighted?: boolean
  disabled?: boolean
  onSelect?: () => void
}) {
  const content = (
    <>
      <span className="text-[10px] font-medium tracking-wide text-muted-foreground uppercase">
        {label}
      </span>
      <span
        className={cn(
          'tabular-nums min-w-11 rounded px-1.5 py-1 text-center text-sm font-medium',
          highlighted
            ? 'border border-primary/40 bg-primary/10 text-primary'
            : 'text-foreground',
        )}
      >
        {value}
      </span>
    </>
  )

  if (!onSelect || disabled) {
    return <div className="flex flex-col items-center gap-0.5">{content}</div>
  }

  return (
    <button
      type="button"
      onClick={onSelect}
      className={cn(
        'flex min-h-11 min-w-11 flex-col items-center justify-center gap-0.5 rounded-md px-1.5 py-1 transition-colors',
        'hover:bg-primary/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        highlighted && 'bg-primary/5',
      )}
      aria-label={`Challenge on ${label}`}
    >
      {content}
    </button>
  )
}

const MemoMatchRow = memo(function MatchRow({
  match,
  showOdds,
  onChallenge,
}: {
  match: Match
  showOdds: boolean
  onChallenge?: (match: Match, side: Side) => void
}) {
  const labels = matchLabels(match)
  const scores = matchScores(match)
  const phase = classifyMatch(match)
  const isLive = phase === 'live'
  const canChallenge = phase === 'upcoming' || phase === 'live'
  const odds = match.odds
  const homeOdds = odds?.home ?? null
  const drawOdds = odds?.draw ?? null
  const awayOdds = odds?.away ?? null
  const lowestOdd =
    homeOdds != null && drawOdds != null && awayOdds != null
      ? Math.min(homeOdds, drawOdds, awayOdds)
      : null

  return (
    <tr className="border-b border-border/70 transition-colors hover:bg-muted/30">
      <td className="w-22 px-3 py-3 align-middle">
        <div className="flex flex-col items-start gap-0.5 tabular-nums">
          <span className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
            {formatKickoffDate(match)}
          </span>
          <span className="text-sm font-semibold text-foreground">
            {formatKickoffClock(match)}
          </span>
          <span
            className={cn(
              'text-xs font-medium',
              match.is_final
                ? 'text-muted-foreground'
                : isLive
                  ? 'text-status-open'
                  : 'text-muted-foreground',
            )}
          >
            {formatStatusShort(match)}
          </span>
        </div>
      </td>

      <td className="min-w-40 px-3 py-3 align-middle">
        <div className="flex flex-col gap-2">
          <TeamRow name={labels.homeTeam} />
          <TeamRow name={labels.awayTeam} />
        </div>
      </td>

      <td className="w-14 px-2 py-3 align-middle">
        <div className="flex flex-col items-center gap-2 tabular-nums">
          <span
            className={cn(
              'text-lg font-semibold leading-none',
              scores.hasScore ? 'text-foreground' : 'text-muted-foreground/50',
            )}
          >
            {scores.hasScore ? scores.home : '—'}
          </span>
          <span
            className={cn(
              'text-lg font-semibold leading-none',
              scores.hasScore ? 'text-foreground' : 'text-muted-foreground/50',
            )}
          >
            {scores.hasScore ? scores.away : '—'}
          </span>
        </div>
      </td>

      {showOdds ? (
        <td className="px-3 py-3 align-middle">
          <div className="flex items-center justify-end gap-2 sm:gap-3">
            <OddsCell
              label="1"
              value={formatOdds(homeOdds)}
              highlighted={lowestOdd != null && homeOdds === lowestOdd}
              disabled={!canChallenge}
              onSelect={
                canChallenge ? () => onChallenge?.(match, 'home') : undefined
              }
            />
            <OddsCell
              label="X"
              value={formatOdds(drawOdds)}
              highlighted={lowestOdd != null && drawOdds === lowestOdd}
              disabled={!canChallenge}
              onSelect={
                canChallenge ? () => onChallenge?.(match, 'draw') : undefined
              }
            />
            <OddsCell
              label="2"
              value={formatOdds(awayOdds)}
              highlighted={lowestOdd != null && awayOdds === lowestOdd}
              disabled={!canChallenge}
              onSelect={
                canChallenge ? () => onChallenge?.(match, 'away') : undefined
              }
            />
          </div>
        </td>
      ) : null}

      <td className="w-12 px-2 py-3 align-middle">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-11 text-muted-foreground hover:text-primary"
          disabled={!canChallenge}
          aria-label={`Challenge ${labels.homeTeam} vs ${labels.awayTeam}`}
          onClick={() => onChallenge?.(match, 'home')}
        >
          <Swords className="size-4" aria-hidden />
        </Button>
      </td>
    </tr>
  )
})

const MemoMatchCard = memo(function MatchCard({
  match,
  showOdds,
  onChallenge,
}: {
  match: Match
  showOdds: boolean
  onChallenge?: (match: Match, side: Side) => void
}) {
  const labels = matchLabels(match)
  const scores = matchScores(match)
  const phase = classifyMatch(match)
  const isLive = phase === 'live'
  const canChallenge = phase === 'upcoming' || phase === 'live'
  const odds = match.odds
  const homeOdds = odds?.home ?? null
  const drawOdds = odds?.draw ?? null
  const awayOdds = odds?.away ?? null
  const lowestOdd =
    homeOdds != null && drawOdds != null && awayOdds != null
      ? Math.min(homeOdds, drawOdds, awayOdds)
      : null

  return (
    <li className="border-b border-border/60 last:border-b-0">
      <div className="flex items-center gap-3 px-3 py-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-baseline justify-between">
            <div className="flex items-baseline gap-2">
              <span className="text-xs font-medium text-foreground tabular-nums">
                {formatKickoffClock(match)}
              </span>
              <span className="text-[10px] font-medium tracking-wide text-muted-foreground uppercase">
                {formatKickoffDate(match)}
              </span>
            </div>
            <span
              className={cn(
                'text-xs font-medium',
                match.is_final
                  ? 'text-muted-foreground'
                  : isLive
                    ? 'text-status-open'
                    : 'text-muted-foreground',
              )}
            >
              {formatStatusShort(match)}
            </span>
          </div>
          <div className="mt-2 flex items-center gap-3">
            <div className="flex flex-1 flex-col gap-1.5">
              <TeamRow name={labels.homeTeam} flagSize="sm" />
              <TeamRow name={labels.awayTeam} flagSize="sm" />
            </div>
            {scores.hasScore ? (
              <div className="flex flex-col items-center gap-1 tabular-nums">
                <span className="text-lg font-semibold leading-none text-foreground">
                  {scores.home}
                </span>
                <span className="text-lg font-semibold leading-none text-foreground">
                  {scores.away}
                </span>
              </div>
            ) : null}
          </div>
        </div>

        {showOdds && canChallenge ? (
          <div className="flex shrink-0 items-center gap-1.5 self-center">
            <OddsCell
              label="1"
              value={formatOdds(homeOdds)}
              highlighted={lowestOdd != null && homeOdds === lowestOdd}
              onSelect={() => onChallenge?.(match, 'home')}
            />
            <OddsCell
              label="X"
              value={formatOdds(drawOdds)}
              highlighted={lowestOdd != null && drawOdds === lowestOdd}
              onSelect={() => onChallenge?.(match, 'draw')}
            />
            <OddsCell
              label="2"
              value={formatOdds(awayOdds)}
              highlighted={lowestOdd != null && awayOdds === lowestOdd}
              onSelect={() => onChallenge?.(match, 'away')}
            />
          </div>
        ) : canChallenge ? (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-10 shrink-0 text-muted-foreground hover:text-primary"
            aria-label={`Challenge ${labels.homeTeam} vs ${labels.awayTeam}`}
            onClick={() => onChallenge?.(match, 'home')}
          >
            <Swords className="size-4" aria-hidden />
          </Button>
        ) : null}
      </div>
    </li>
  )
})

function DesktopCompetitionSection({
  competition,
  matches,
  showOdds,
  onChallenge,
}: {
  competition: string
  matches: Match[]
  showOdds: boolean
  onChallenge?: (match: Match, side: Side) => void
}) {
  return (
    <div className="overflow-hidden rounded-lg border border-border bg-card shadow-sahara">
      <div className="flex items-center gap-2 border-b border-border bg-muted/40 px-4 py-3">
        <Trophy className="size-4 shrink-0 text-primary" aria-hidden />
        <div className="min-w-0">
          <h3 className="truncate font-heading text-lg leading-tight text-foreground">
            {competition}
          </h3>
          <p className="text-xs text-muted-foreground">
            {matches.length} {matches.length === 1 ? 'fixture' : 'fixtures'}
          </p>
        </div>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full min-w-lg border-collapse text-sm">
          <caption className="sr-only">{competition} fixtures</caption>
          <thead>
            <tr className="border-b border-border text-left text-xs text-muted-foreground">
              <th scope="col" className="px-3 py-2 font-medium">
                Date · Time
              </th>
              <th scope="col" className="px-3 py-2 font-medium">
                Teams
              </th>
              <th scope="col" className="px-2 py-2 text-center font-medium">
                Score
              </th>
              {showOdds ? (
                <th scope="col" className="px-3 py-2 text-right font-medium">
                  1 · X · 2
                </th>
              ) : null}
              <th scope="col" className="w-12 px-2 py-2">
                <span className="sr-only">Challenge</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {matches.map((match) => (
              <MemoMatchRow
                key={match.match_id}
                match={match}
                showOdds={showOdds}
                onChallenge={onChallenge}
              />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function MobileCompetitionSection({
  competition,
  matches,
  showOdds,
  onChallenge,
}: {
  competition: string
  matches: Match[]
  showOdds: boolean
  onChallenge?: (match: Match, side: Side) => void
}) {
  return (
    <div className="overflow-hidden rounded-lg border border-border bg-card shadow-sahara">
      <div className="flex items-center gap-2 border-b border-border bg-muted/40 px-4 py-3">
        <Trophy className="size-4 shrink-0 text-primary" aria-hidden />
        <div className="min-w-0">
          <h3 className="truncate font-heading text-lg leading-tight text-foreground">
            {competition}
          </h3>
          <p className="text-xs text-muted-foreground">
            {matches.length} {matches.length === 1 ? 'fixture' : 'fixtures'}
          </p>
        </div>
      </div>
      <ul className="divide-y divide-border/60">
        {matches.map((match) => (
          <MemoMatchCard
            key={match.match_id}
            match={match}
            showOdds={showOdds}
            onChallenge={onChallenge}
          />
        ))}
      </ul>
    </div>
  )
}

export function MatchTable() {
  const { data: matches, isError, error, isLoading } = useMatchesQuery()
  const isDesktop = useMediaQuery('(min-width: 40rem)')
  const [activeFilter, setActiveFilter] = useState<MatchFilter>('live')
  const [filterTouched, setFilterTouched] = useState(false)
  const [showOdds, setShowOdds] = useState(true)
  const [slipOpen, setSlipOpen] = useState(false)
  const [slipMatch, setSlipMatch] = useState<Match | null>(null)
  const [slipSide, setSlipSide] = useState<Side>('home')

  const openChallenge = (match: Match, side: Side) => {
    setSlipMatch(match)
    setSlipSide(side)
    setSlipOpen(true)
  }

  const counts = useMemo(
    () => countMatchesByFilter(matches ?? []),
    [matches],
  )

  const filtered = useMemo(() => {
    if (!matches?.length) return []
    return filterMatches(matches, activeFilter)
  }, [matches, activeFilter])

  const defaultFilter = useMemo<MatchFilter>(() => {
    const tally = countMatchesByFilter(matches ?? [])
    if (tally.live > 0) return 'live'
    if (tally.finished > 0) return 'finished'
    return 'upcoming'
  }, [matches])

  useEffect(() => {
    if (!filterTouched && matches?.length) {
      setActiveFilter(defaultFilter)
    }
  }, [matches, defaultFilter, filterTouched])

  const groups = useMemo(
    () => groupMatchesByCompetition(filtered),
    [filtered],
  )

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex gap-2 pb-1">
            <Skeleton className="h-11 w-24 rounded-full" />
            <Skeleton className="h-11 w-24 rounded-full" />
            <Skeleton className="h-11 w-28 rounded-full" />
          </div>
          <div className="flex items-center justify-end gap-2">
            <Skeleton className="h-4 w-8" />
            <Skeleton className="h-6 w-11 rounded-full" />
          </div>
        </div>

        <div className="space-y-4">
          {[1, 2].map((i) => (
            <div key={i} className="overflow-hidden rounded-lg border border-border bg-card shadow-sahara">
              <div className="flex items-center gap-2 border-b border-border bg-muted/40 px-4 py-3">
                <Skeleton className="size-4 rounded-full shrink-0" />
                <div className="min-w-0 space-y-1.5">
                  <Skeleton className="h-5 w-32" />
                  <Skeleton className="h-3 w-16" />
                </div>
              </div>
              <div className="divide-y divide-border/60">
                {[1, 2, 3].map((j) => (
                  <div key={j} className="flex items-center gap-3 px-3 py-3">
                    <div className="min-w-0 flex-1">
                      <div className="flex items-baseline justify-between mb-3">
                        <Skeleton className="h-3 w-20" />
                        <Skeleton className="h-3 w-8" />
                      </div>
                      <div className="flex flex-col gap-2">
                        <Skeleton className="h-4 w-48 max-w-[80%]" />
                        <Skeleton className="h-4 w-40 max-w-[70%]" />
                      </div>
                    </div>
                    <div className="hidden sm:flex shrink-0 items-center gap-1.5 self-center">
                      <Skeleton className="h-11 w-11 rounded-md" />
                      <Skeleton className="h-11 w-11 rounded-md" />
                      <Skeleton className="h-11 w-11 rounded-md" />
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (isError) {
    return (
      <p className="text-sm text-destructive">
        {error instanceof Error ? error.message : 'Failed to load matches'}
      </p>
    )
  }

  const SectionComponent = isDesktop ? DesktopCompetitionSection : MobileCompetitionSection

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div
          className="flex gap-2 overflow-x-auto pb-1"
          role="tablist"
          aria-label="Match status filters"
        >
          {FILTERS.map(({ id, label }) => {
            const selected = activeFilter === id
            const count = counts[id]
            return (
              <button
                key={id}
                type="button"
                role="tab"
                aria-selected={selected}
                onClick={() => {
                  setFilterTouched(true)
                  setActiveFilter(id)
                }}
                className={cn(
                  'shrink-0 rounded-full px-4 py-2 text-sm font-medium transition-colors',
                  'min-h-11 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                  selected
                    ? id === 'live'
                      ? 'bg-destructive text-white'
                      : 'bg-foreground text-background'
                    : 'bg-muted text-muted-foreground hover:bg-muted/80',
                )}
              >
                {label}
                {count > 0 ? ` (${count})` : ''}
              </button>
            )
          })}
        </div>

        <label className="flex min-h-11 cursor-pointer items-center justify-end gap-2 text-sm text-muted-foreground">
          <span>Odds</span>
          <button
            type="button"
            role="switch"
            aria-checked={showOdds}
            onClick={() => setShowOdds((value) => !value)}
            className={cn(
              'relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
              showOdds ? 'bg-primary' : 'bg-muted',
            )}
          >
            <span
              className={cn(
                'inline-block size-4 rounded-full bg-white shadow transition-transform',
                showOdds ? 'translate-x-6' : 'translate-x-1',
              )}
            />
          </button>
        </label>
      </div>

      {!matches?.length ? (
        <div className="rounded-lg border border-dashed bg-muted/40 px-6 py-12 text-center">
          <p className="font-heading text-2xl">No active markets</p>
          <p className="mx-auto mt-2 max-w-sm text-sm text-muted-foreground">
            Upcoming fixtures load from the TxLINE schedule snapshot; live scores
            update via SSE once the keeper is running.
          </p>
        </div>
      ) : filtered.length === 0 ? (
        <div className="rounded-lg border border-dashed bg-muted/40 px-6 py-10 text-center">
          <p className="font-heading text-xl">No {activeFilter} matches</p>
          <p className="mx-auto mt-2 max-w-sm text-sm text-muted-foreground">
            Try another filter to browse fixtures in this competition window.
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {groups.map((group) => (
            <SectionComponent
              key={group.competition}
              competition={group.competition}
              matches={group.matches}
              showOdds={showOdds}
              onChallenge={openChallenge}
            />
          ))}
        </div>
      )}

      {slipMatch ? (
        <ChallengeSlipDialog
          match={slipMatch}
          initialSide={slipSide}
          open={slipOpen}
          onOpenChange={setSlipOpen}
        />
      ) : null}
    </div>
  )
}