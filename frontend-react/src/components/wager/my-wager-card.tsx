import { CheckCircle, Clock, Swords, User } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter } from '@/components/ui/card'
import { DuelFrame } from '@/components/wager/duel-frame'
import { SettlementStatus } from '@/components/wager/settlement-status'
import { WagerStatusBadge } from '@/components/wager/wager-status-badge'
import { useStablecoinLabel } from '@/hooks/use-stablecoin-label'
import type { Match, Wager } from '@/lib/api'
import { formatStakeBaseUnits, truncateAddress } from '@/lib/format'
import { matchLabels } from '@/lib/match-display'
import { sideLabel } from '@/lib/wager-sides'
import { isPlaceholderAddress } from '@/lib/accounts'
import { cn } from '@/lib/utils'

export type MyWagerCardProps = {
  wager: Wager
  match?: Match
  walletAddress: string
  onSelect?: () => void
  onCancel?: () => void
  onClaim?: () => void
  claimable?: boolean
  claimPending?: boolean
  claimed?: boolean
  className?: string
}

function roleLabel(wager: Wager, isMaker: boolean) {
  if (wager.status === 'cancelled') {
    return isMaker ? 'You cancelled this wager' : 'This wager was cancelled'
  }
  if (wager.status === 'open') {
    return isMaker ? 'Your open challenge' : 'Challenge you joined'
  }
  return isMaker ? 'You posted this wager' : 'You accepted this wager'
}

export function MyWagerCard({
  wager,
  match,
  walletAddress,
  onSelect,
  onCancel,
  onClaim,
  claimable = false,
  claimPending = false,
  claimed = false,
  className,
}: MyWagerCardProps) {
  const labels = match ? matchLabels(match) : null
  const stablecoin = useStablecoinLabel()
  const isMaker = wager.maker === walletAddress
  const canCancel = isMaker && wager.status === 'open'
  const stake = formatStakeBaseUnits(wager.stake)
  const payout = formatStakeBaseUnits(wager.stake * 2)

  const backedSide = isMaker
    ? wager.maker_side
    : wager.taker_side ?? wager.maker_side
  const backedLabel = match
    ? sideLabel(backedSide, match)
    : backedSide

  const opponent = isMaker ? wager.taker : wager.maker
  const awaitingOpponent =
    wager.status === 'open' && isPlaceholderAddress(opponent)
  const showOpponent = !awaitingOpponent && opponent.length > 0

  return (
    <Card
      className={cn('overflow-hidden', claimed && 'opacity-50 pointer-events-none', onSelect && 'cursor-pointer transition-colors hover:bg-muted/20', className)}
      onClick={onSelect}
    >
      <CardContent className="space-y-3 px-4 py-4">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 space-y-0.5">
            <p className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
              <Swords className="size-3 shrink-0" aria-hidden />
              {roleLabel(wager, isMaker)}
            </p>
          </div>
          <WagerStatusBadge status={wager.status} />
        </div>

        {labels ? (
          <>
            <div className="flex min-w-0 items-center gap-2 text-xs text-muted-foreground">
              <span className="truncate">{labels.league}</span>
              {labels.isLive ? (
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
              ) : labels.isStatusStale ? (
                <>
                  <span aria-hidden>·</span>
                  <span className="inline-flex shrink-0 items-center gap-1 font-medium text-muted-foreground">
                    Result pending
                  </span>
                </>
              ) : null}
            </div>

            <DuelFrame
              home={labels.homeTeam}
              away={labels.awayTeam}
              scoreLine={labels.scoreLine}
              size="dense"
              layout="inline"
            />
          </>
        ) : (
          <p className="font-heading text-base">Match {wager.match_id}</p>
        )}

        <div className="grid gap-2 rounded-md bg-muted/40 px-3 py-2.5 text-xs">
          <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
            <span className="text-muted-foreground">
              Your pick{' '}
              <span className="font-medium text-foreground">{backedLabel}</span>
            </span>
            <span className="tabular-nums text-foreground">
              <span className="font-medium">{stake}</span>{' '}
              <span className="text-muted-foreground">{stablecoin}</span>
            </span>
          </div>
          {wager.status !== 'open' ? (
            <div className="flex items-center justify-between border-t border-border/50 pt-2">
              <span className="text-muted-foreground">If you win</span>
              <span className="tabular-nums font-semibold text-primary">
                {payout} {stablecoin}
              </span>
            </div>
          ) : null}
        </div>

        {awaitingOpponent ? (
          <p className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
            <Clock className="size-3 shrink-0" aria-hidden />
            Waiting for an opponent to accept your challenge
          </p>
        ) : showOpponent ? (
          <p className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
            <User className="size-3 shrink-0" aria-hidden />
            Opponent{' '}
            <span className="font-mono">{truncateAddress(opponent)}</span>
          </p>
        ) : null}

        {wager.status === 'matched' ? (
          <SettlementStatus wagerPubkey={wager.pubkey} match={match} />
        ) : null}
      </CardContent>

      {canCancel || claimable || claimed ? (
        <CardFooter className="border-t px-4 py-3">
          {claimed ? (
            <Button
              className="min-h-10 w-full"
              disabled
            >
              <CheckCircle className="mr-1.5 size-4 shrink-0 text-green-600" aria-hidden />
              <span className="text-green-600">Claimed</span>
            </Button>
          ) : claimable ? (
            <Button
              className="min-h-10 w-full"
              disabled={claimPending}
              onClick={(e) => { e.stopPropagation(); onClaim?.() }}
            >
              Claim winnings
            </Button>
          ) : (
            <Button
              variant="outline"
              className="min-h-10 w-full"
              onClick={(e) => { e.stopPropagation(); onCancel?.() }}
            >
              Cancel wager
            </Button>
          )}
        </CardFooter>
      ) : null}
    </Card>
  )
}
