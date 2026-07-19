import { UserPlus } from 'lucide-react'
import { useEffect, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter } from '@/components/ui/card'
import { DuelFrame } from '@/components/wager/duel-frame'
import { OutcomePicker } from '@/components/wager/outcome-picker'
import { WagerStatusBadge } from '@/components/wager/wager-status-badge'
import type { Match, Side, Wager } from '@/lib/api'
import { useStablecoinLabel } from '@/hooks/use-stablecoin-label'
import { formatStakeBaseUnits, truncateAddress } from '@/lib/format'
import { matchLabels } from '@/lib/match-display'
import {
  availableTakerSides,
  defaultTakerSide,
  sideLabel,
} from '@/lib/wager-sides'
import { cn } from '@/lib/utils'

export interface OpenWagerCardProps {
  wager: Wager
  match?: Match
  onAccept?: (takerSide: Side) => void
  disabled?: boolean
  accepted?: boolean
  className?: string
}

export function OpenWagerCard({
  wager,
  match,
  onAccept,
  disabled = false,
  accepted = false,
  className,
}: OpenWagerCardProps) {
  const labels = match ? matchLabels(match) : null
  const homeTeam = labels?.homeTeam ?? 'Home'
  const awayTeam = labels?.awayTeam ?? 'Away'
  const league = labels?.league ?? `Match ${wager.match_id}`
  const stablecoin = useStablecoinLabel()
  const stake = formatStakeBaseUnits(wager.stake)
  const payout = formatStakeBaseUnits(wager.stake * 2)

  const takerOptions = availableTakerSides(wager.maker_side)
  const [takerSide, setTakerSide] = useState<Side>(
    defaultTakerSide(wager.maker_side),
  )

  useEffect(() => {
    setTakerSide(defaultTakerSide(wager.maker_side))
  }, [wager.pubkey, wager.maker_side])

  const makerSideLabel = match
    ? sideLabel(wager.maker_side, match)
    : wager.maker_side

  return (
    <Card className={cn('overflow-hidden', accepted && 'opacity-50 pointer-events-none', className)}>
      <CardContent className="space-y-3 px-4 pt-4 pb-0">
        <div className="flex items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-2 text-xs text-muted-foreground">
            <span className="truncate">{league}</span>
            {labels?.isLive ? (
              <>
                <span aria-hidden>·</span>
                <span className="inline-flex shrink-0 items-center gap-1 font-medium text-status-open">
                  <span
                    className="size-1.5 rounded-full bg-status-open motion-safe:animate-pulse"
                    aria-hidden
                  />
                  Live
                </span>
              </>
            ) : labels?.isStatusStale ? (
              <>
                <span aria-hidden>·</span>
                <span className="inline-flex shrink-0 items-center gap-1 font-medium text-muted-foreground">
                  Result pending
                </span>
              </>
            ) : null}
          </div>
          <WagerStatusBadge status={wager.status} />
        </div>

        <DuelFrame
          home={homeTeam}
          away={awayTeam}
          size="dense"
          layout="inline"
        />

        <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 rounded-md bg-muted/40 px-3 py-2.5 text-xs">
          <span className="text-muted-foreground">
            They pick{' '}
            <span className="font-medium text-foreground">{makerSideLabel}</span>
          </span>
          <span className="tabular-nums text-foreground">
            <span className="font-medium">{stake}</span>{' '}
            <span className="text-muted-foreground">{stablecoin}</span>
            <span className="mx-1.5 text-muted-foreground" aria-hidden>
              →
            </span>
            <span className="font-semibold text-primary">{payout}</span>{' '}
            <span className="text-muted-foreground">win</span>
          </span>
        </div>

        <OutcomePicker
          match={match}
          sides={takerOptions}
          selected={takerSide}
          onSelect={setTakerSide}
          label="Your outcome"
          showOdds={Boolean(match)}
          density="compact"
        />

        <p className="flex items-center gap-1.5 pb-1 text-[11px] text-muted-foreground">
          <UserPlus className="size-3 shrink-0" aria-hidden />
          Posted by{' '}
          <span className="font-mono">{truncateAddress(wager.maker)}</span>
        </p>
      </CardContent>

      <CardFooter className="border-t px-4 py-3">
        <Button
          className="min-h-10 w-full"
          disabled={disabled || accepted || Boolean(labels?.isStatusStale)}
          onClick={() => onAccept?.(takerSide)}
        >
          {accepted ? 'Accepted' : 'Accept challenge'}
        </Button>
      </CardFooter>
    </Card>
  )
}
