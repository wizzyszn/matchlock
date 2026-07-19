import { Clock, ExternalLink, Loader2, Trophy } from 'lucide-react'
import { useMemo } from 'react'

import { useWagerSettlementQuery } from '@/hooks/queries/use-wagers'
import { useConfig } from '@/hooks/use-api'
import type { Match, SettlementState } from '@/lib/api'
import { explorerTxUrl } from '@/lib/format'
import { cn } from '@/lib/utils'

type SettlementStatusProps = {
  wagerPubkey: string
  match?: Match
  lastSignature?: string | null
}

function matchNotStarted(match?: Match): boolean {
  if (!match?.start_time || match.start_time <= 0) return false
  return Date.now() < match.start_time
}

const STATE_COPY: Record<
  Exclude<SettlementState, 'not_applicable'>,
  (preMatch?: boolean) => { title: string; detail: string }
> = {
  match_live: (preMatch) =>
    preMatch
      ? {
          title: 'Match not started yet',
          detail: "The fixture hasn't kicked off. Payout status updates once the result is confirmed.",
        }
      : {
          title: 'Match in progress',
          detail: 'Payout status updates once the final result is confirmed.',
        },
  match_ended_unverified: () => ({
    title: 'Confirming final score',
    detail: 'Payout starts as soon as the official result is verified.',
  }),
  claimable: () => ({
    title: 'Ready to claim',
    detail: 'The final result is verified. The winner can claim the payout.',
  }),
  refundable: () => ({
    title: 'Refund due',
    detail: 'Neither selected outcome won. Both stakes will be returned.',
  }),
  queued: () => ({
    title: 'Settlement queued',
    detail: 'The keeper has scheduled the payout for on-chain processing.',
  }),
  retrying: () => ({
    title: 'Settlement in progress',
    detail: 'The final result is verified. Settlement is being processed on-chain.',
  }),
  failed: () => ({
    title: 'Settlement in progress',
    detail: "This is taking a bit longer than usual — we're still working on it.",
  }),
  settled: () => ({
    title: 'Resolved',
    detail: 'Escrow funds have been released on-chain.',
  }),
}

function toneForState(state: SettlementState) {
  switch (state) {
    case 'settled':
      return 'border-status-settled/25 bg-status-settled-bg text-status-settled'
    case 'match_live':
    case 'match_ended_unverified':
      return 'border-border/60 bg-muted/30 text-muted-foreground'
    default:
      return 'border-primary/20 bg-primary/5 text-foreground'
  }
}

export function SettlementStatus({
  wagerPubkey,
  match,
  lastSignature,
}: SettlementStatusProps) {
  const config = useConfig()
  let { data: status, isLoading, isError } = useWagerSettlementQuery(wagerPubkey)
  const preMatch = useMemo(() => matchNotStarted(match), [match])

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 rounded-md border border-border/60 bg-muted/30 px-3 py-2.5 text-xs text-muted-foreground">
        <Loader2 className="size-3.5 animate-spin" aria-hidden />
        Checking payout status…
      </div>
    )
  }

  if (isError || !status || status.state === 'not_applicable') {
    return null
  }

  const getCopy = STATE_COPY[status.state]
  const copy = getCopy(preMatch)
  const title = copy.title
  const detail = copy.detail || status.message 
  const inProgress =
    status.state === 'queued' ||
    status.state === 'retrying' ||
    status.state === 'failed'
  const Icon =
    status.state === 'settled'
      ? Trophy
      : inProgress
        ? Loader2
        : Clock
  const signature = status.tx_signature || lastSignature

  return (
    <div
      className={cn(
        'flex items-start gap-2.5 rounded-md border px-3 py-2.5 text-sm',
        toneForState(status.state),
      )}
      role="status"
      aria-live="polite"
    >
      <Icon
        className={cn(
          'mt-0.5 size-4 shrink-0',
          inProgress && 'motion-safe:animate-spin',
        )}
        aria-hidden
      />
      <div className="min-w-0 space-y-0.5">
        <p className="font-medium leading-snug">{title}</p>
        <p className="text-xs leading-relaxed opacity-90">{detail}</p>
        {signature && signature !== 'already-settled' ? (
          <a
            href={explorerTxUrl(signature, config.cluster)}
            target="_blank"
            rel="noreferrer"
            className="mt-1 inline-flex items-center gap-1 text-xs font-medium underline-offset-4 hover:underline"
          >
            View transaction
            <ExternalLink className="size-3" aria-hidden />
          </a>
        ) : null}
      </div>
    </div>
  )
}
