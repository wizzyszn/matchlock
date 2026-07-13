import { Trophy } from 'lucide-react'
import { useEffect, useMemo, useRef } from 'react'

import { TeamFlag } from '@/components/wager/team-flag'
import type { Match } from '@/lib/api'
import {
  classifyMatch,
  formatKickoffClock,
  formatKickoffDate,
  formatStatusShort,
  groupMatchesByCompetition,
  matchLabels,
  matchScores,
} from '@/lib/match-display'
import { cn } from '@/lib/utils'

export type FixturePickerProps = {
  matches: Match[]
  selectedId: string
  onSelect: (matchId: string) => void
  className?: string
}

function FixtureCard({
  match,
  selected,
  onSelect,
}: {
  match: Match
  selected: boolean
  onSelect: () => void
}) {
  const labels = matchLabels(match)
  const scores = matchScores(match)
  const phase = classifyMatch(match)
  const isLive = phase === 'live'
  const statusShort = formatStatusShort(match)
  const cardRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (selected) {
      cardRef.current?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
    }
  }, [selected])

  return (
    <button
      ref={cardRef}
      type="button"
      role="radio"
      aria-checked={selected}
      onClick={onSelect}
      className={cn(
        'w-full rounded-lg border bg-background p-3 text-left transition-colors',
        'min-h-11 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        selected
          ? 'border-primary bg-primary/5 shadow-sm'
          : 'border-border hover:border-primary/35 hover:bg-muted/40',
      )}
    >
      <div className="flex items-center justify-between gap-2 text-[11px] text-muted-foreground">
        <span className="tabular-nums">
          {formatKickoffDate(match)} · {formatKickoffClock(match)}
        </span>
        {statusShort !== '—' ? (
          <span
            className={cn(
              'inline-flex items-center gap-1 font-medium',
              match.is_final
                ? 'text-muted-foreground'
                : isLive
                  ? 'text-status-open'
                  : 'text-muted-foreground',
            )}
          >
            {isLive ? (
              <span
                className="size-1.5 rounded-full bg-status-open motion-safe:animate-pulse"
                aria-hidden
              />
            ) : null}
            {statusShort}
          </span>
        ) : null}
      </div>

      <div className="mt-2.5 grid grid-cols-[1fr_auto_1fr] items-center gap-2">
        <div className="flex min-w-0 flex-col items-center gap-1.5 text-center">
          <TeamFlag name={labels.homeTeam} size="md" />
          <span className="line-clamp-2 font-heading text-sm leading-tight text-foreground">
            {labels.homeTeam}
          </span>
        </div>

        <div className="flex flex-col items-center gap-0.5 px-1 tabular-nums">
          {scores.hasScore ? (
            <>
              <span className="text-base font-semibold leading-none text-foreground">
                {scores.home}
              </span>
              <span className="text-[10px] font-medium tracking-widest text-muted-foreground uppercase">
                –
              </span>
              <span className="text-base font-semibold leading-none text-foreground">
                {scores.away}
              </span>
            </>
          ) : (
            <span className="text-xs font-medium tracking-widest text-muted-foreground uppercase">
              vs
            </span>
          )}
        </div>

        <div className="flex min-w-0 flex-col items-center gap-1.5 text-center">
          <TeamFlag name={labels.awayTeam} size="md" />
          <span className="line-clamp-2 font-heading text-sm leading-tight text-foreground">
            {labels.awayTeam}
          </span>
        </div>
      </div>
    </button>
  )
}

export function FixturePicker({
  matches,
  selectedId,
  onSelect,
  className,
}: FixturePickerProps) {
  const groups = useMemo(
    () => groupMatchesByCompetition(matches),
    [matches],
  )

  return (
    <div
      role="radiogroup"
      aria-label="Choose a fixture"
      className={cn('space-y-4', className)}
    >
      {groups.map(({ competition, matches: groupMatches }) => (
        <section key={competition} className="space-y-2">
          <div className="flex items-center gap-2 px-0.5">
            <Trophy className="size-3.5 shrink-0 text-primary" aria-hidden />
            <h3 className="truncate font-heading text-base leading-tight text-foreground">
              {competition}
            </h3>
            <span className="shrink-0 text-xs text-muted-foreground">
              {groupMatches.length}
            </span>
          </div>

          <div className="max-h-[min(22rem,50vh)] space-y-2 overflow-y-auto pr-0.5 [-ms-overflow-style:none] [scrollbar-width:thin] [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-border">
            {groupMatches.map((match) => (
              <FixtureCard
                key={match.match_id}
                match={match}
                selected={match.match_id === selectedId}
                onSelect={() => onSelect(match.match_id)}
              />
            ))}
          </div>
        </section>
      ))}
    </div>
  )
}
