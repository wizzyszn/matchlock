import { Check } from 'lucide-react'

import { TeamFlag } from '@/components/wager/team-flag'
import type { Match, Side } from '@/lib/api'
import { formatOdds, matchLabels } from '@/lib/match-display'
import { referenceOddsForSide, sideShortLabel } from '@/lib/wager-sides'
import { cn } from '@/lib/utils'

export type OutcomeDensity = 'default' | 'compact'

export type OutcomeOptionProps = {
  side: Side
  match?: Match
  selected: boolean
  onSelect: () => void
  showOdds?: boolean
  density?: OutcomeDensity
}

export function OutcomeOption({
  side,
  match,
  selected,
  onSelect,
  showOdds = true,
  density = 'default',
}: OutcomeOptionProps) {
  const compact = density === 'compact'
  const labels = match ? matchLabels(match) : null
  const odds = match && showOdds ? referenceOddsForSide(side, match.odds) : null
  const isDraw = side === 'draw'
  const displayName = match
    ? side === 'home'
      ? labels!.homeTeam
      : side === 'away'
        ? labels!.awayTeam
        : 'Draw'
    : side === 'draw'
      ? 'Draw'
      : side === 'home'
        ? 'Home'
        : 'Away'

  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      onClick={onSelect}
      className={cn(
        'flex w-full items-center text-left transition-colors',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        compact
          ? 'min-h-11 gap-2 rounded-md border px-2.5 py-1.5'
          : 'min-h-14 gap-3 rounded-lg border px-3 py-2.5',
        selected
          ? 'border-primary bg-primary/8 shadow-sm'
          : 'border-border bg-background hover:border-primary/35 hover:bg-muted/40',
      )}
    >
      <span
        className={cn(
          'flex shrink-0 items-center justify-center rounded-md border border-border bg-muted/50 font-bold tracking-widest text-muted-foreground uppercase',
          compact ? 'size-6 text-[10px]' : 'size-8 text-[11px]',
        )}
      >
        {sideShortLabel(side)}
      </span>

      {isDraw ? (
        <span
          className={cn(
            'flex shrink-0 items-center justify-center rounded-full border border-dashed border-muted-foreground/50 font-bold text-muted-foreground',
            compact ? 'size-6 text-[10px]' : 'size-8 text-xs',
          )}
          aria-hidden
        >
          =
        </span>
      ) : match ? (
        <TeamFlag
          name={displayName}
          size={compact ? 'sm' : 'md'}
          className="shrink-0"
        />
      ) : (
        <span
          className={cn(
            'flex shrink-0 items-center justify-center rounded-full border border-border bg-muted font-semibold text-muted-foreground',
            compact ? 'size-6 text-[10px]' : 'size-8 text-xs',
          )}
          aria-hidden
        >
          {sideShortLabel(side)}
        </span>
      )}

      <div className="min-w-0 flex-1">
        <p
          className={cn(
            'truncate text-foreground',
            compact
              ? 'text-sm font-medium'
              : isDraw
                ? 'font-medium'
                : 'font-heading text-base leading-tight',
          )}
        >
          {displayName}
        </p>
        {!compact ? (
          <p className="mt-0.5 text-xs text-muted-foreground">
            {isDraw
              ? 'Neither side wins in regular time'
              : `You win if ${displayName} takes the match`}
          </p>
        ) : null}
      </div>

      <div
        className={cn(
          'flex shrink-0 items-center',
          compact ? 'gap-2' : 'flex-col items-end justify-center gap-1 self-stretch py-0.5',
        )}
      >
        {showOdds && match ? (
          odds != null ? (
            <span
              className={cn(
                'tabular-nums font-semibold text-foreground',
                compact ? 'text-xs' : 'text-sm',
              )}
            >
              {formatOdds(odds)}
            </span>
          ) : (
            <span className={compact ? 'text-xs text-muted-foreground' : 'text-sm text-muted-foreground'}>
              —
            </span>
          )
        ) : null}
        {selected ? (
          <Check className={compact ? 'size-3.5 text-primary' : 'size-4 text-primary'} aria-hidden />
        ) : compact ? null : (
          <span className="size-4" aria-hidden />
        )}
      </div>
    </button>
  )
}

export type OutcomePickerProps = {
  match?: Match
  sides: Side[]
  selected: Side
  onSelect: (side: Side) => void
  label?: string
  hint?: string
  showOdds?: boolean
  density?: OutcomeDensity
  className?: string
}

export function OutcomePicker({
  match,
  sides,
  selected,
  onSelect,
  label = 'Pick your outcome',
  hint,
  showOdds = true,
  density = 'default',
  className,
}: OutcomePickerProps) {
  const compact = density === 'compact'

  return (
    <div className={cn(compact ? 'space-y-1.5' : 'space-y-2', className)}>
      <span className={cn('font-medium', compact ? 'text-xs text-muted-foreground' : 'text-sm')}>
        {label}
      </span>
      <div
        role="radiogroup"
        aria-label={label}
        className={cn(compact ? 'space-y-1.5' : 'space-y-2')}
      >
        {sides.map((side) => (
          <OutcomeOption
            key={side}
            side={side}
            match={match}
            selected={selected === side}
            onSelect={() => onSelect(side)}
            showOdds={showOdds}
            density={density}
          />
        ))}
      </div>
      {hint ? (
        <p className="text-xs text-muted-foreground">{hint}</p>
      ) : null}
    </div>
  )
}