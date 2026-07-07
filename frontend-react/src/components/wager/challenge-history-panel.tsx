import { useWallet } from '@solana/wallet-adapter-react'
import { CircleX, Crown, Loader2, Skull, Swords, User } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { DuelFrame } from '@/components/wager/duel-frame'
import { useMatchesQuery } from '@/hooks/queries/use-matches'
import { useWagersQuery } from '@/hooks/queries/use-wagers'
import { useStablecoinLabel } from '@/hooks/use-stablecoin-label'
import type { Match, Wager } from '@/lib/api'
import { formatStakeBaseUnits, truncateAddress } from '@/lib/format'
import { matchLabels } from '@/lib/match-display'
import { cn } from '@/lib/utils'
import { userBackedSide } from '@/lib/wager-outcome'
import { sideLabel } from '@/lib/wager-sides'
import { isPlaceholderAddress } from '@/lib/accounts'

type OutcomeFilter = 'all' | 'won' | 'lost' | 'void'

type HistoryEntry = {
  wager: Wager
  match?: Match
  outcome: 'won' | 'lost' | 'void'
  backedLabel: string
  stakeFmt: string
  opponent: string
  isMaker: boolean
  date: number
}

const FILTER_OPTIONS: { value: OutcomeFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'won', label: 'Won' },
  { value: 'lost', label: 'Lost' },
  { value: 'void', label: 'Void' },
]

function outcomeLabel(outcome: 'won' | 'lost' | 'void') {
  switch (outcome) {
    case 'won':
      return 'Won'
    case 'lost':
      return 'Lost'
    case 'void':
      return 'Void'
  }
}

function outcomeIcon(outcome: 'won' | 'lost' | 'void') {
  switch (outcome) {
    case 'won':
      return Crown
    case 'lost':
      return Skull
    case 'void':
      return CircleX
  }
}

function outcomeBadgeClass(outcome: 'won' | 'lost' | 'void') {
  switch (outcome) {
    case 'won':
      return 'border-status-settled/25 bg-status-settled-bg text-status-settled'
    case 'lost':
      return 'border-status-cancelled/25 bg-status-cancelled-bg text-status-cancelled'
    case 'void':
      return 'border-muted-foreground/25 bg-muted/40 text-muted-foreground'
  }
}

function winningSideFromMatch(match: Match): 'home' | 'draw' | 'away' | null {
  if (!match.is_final) return null
  const home = match.home_goals
  const away = match.away_goals
  if (home == null || away == null) return null
  if (home > away) return 'home'
  if (away > home) return 'away'
  if (home === away) return 'draw'
  return null
}

function classifyOutcome(
  wager: Wager,
  match: Match | undefined,
  walletAddress: string,
): 'won' | 'lost' | 'void' | null {
  if (wager.status === 'cancelled') return 'void'
  if (wager.status !== 'settled' || !match) return null
  const outcome = winningSideFromMatch(match)
  if (!outcome) return null
  return userBackedSide(wager, walletAddress) === outcome ? 'won' : 'lost'
}

function formatDate(ms: number): string {
  if (ms <= 0) return ''
  return new Intl.DateTimeFormat('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  }).format(new Date(ms))
}

export function ChallengeHistoryPanel() {
  const { publicKey } = useWallet()
  const walletAddress = publicKey?.toBase58()
  const navigate = useNavigate()
  const stablecoin = useStablecoinLabel()

  const { data: wagers, isLoading: wagersLoading } = useWagersQuery({
    wallet: walletAddress ?? undefined,
  })
  const { data: matches, isLoading: matchesLoading } = useMatchesQuery()

  const [outcomeFilter, setOutcomeFilter] = useState<OutcomeFilter>('all')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')

  const matchMap = useMemo(
    () => new Map(matches?.map((m) => [m.match_id, m]) ?? []),
    [matches],
  )

  const entries = useMemo<HistoryEntry[]>(() => {
    if (!wagers || !walletAddress) return []

    const raw: HistoryEntry[] = []

    for (const w of wagers) {
      const isMaker = w.maker === walletAddress
      const isTaker = w.taker === walletAddress
      if (!isMaker && !isTaker) continue

      if (w.status !== 'settled' && w.status !== 'cancelled') continue

      const match = matchMap.get(w.match_id)
      const outcome = classifyOutcome(w, match, walletAddress)
      if (!outcome) continue

      const backed = userBackedSide(w, walletAddress)
      const backedLabel = match ? sideLabel(backed, match) : backed

      const opponent = isMaker ? w.taker : w.maker

      const date = match?.start_time ?? 0

      raw.push({
        wager: w,
        match,
        outcome,
        backedLabel,
        stakeFmt: formatStakeBaseUnits(w.stake),
        opponent,
        isMaker,
        date,
      })
    }

    return raw
  }, [wagers, walletAddress, matchMap])

  const filtered = useMemo(() => {
    let result = entries

    if (outcomeFilter !== 'all') {
      result = result.filter((e) => e.outcome === outcomeFilter)
    }

    if (dateFrom) {
      const fromMs = new Date(dateFrom).getTime()
      result = result.filter((e) => e.date >= fromMs)
    }
    if (dateTo) {
      const toMs = new Date(dateTo).getTime() + 86_400_000
      result = result.filter((e) => e.date <= toMs)
    }

    return result.sort((a, b) => b.date - a.date)
  }, [entries, outcomeFilter, dateFrom, dateTo])

  const isLoading = wagersLoading || matchesLoading

  if (!walletAddress) {
    return (
      <div className="rounded-lg border border-dashed bg-muted/40 px-6 py-12 text-center">
        <p className="font-heading text-2xl">Challenge history</p>
        <p className="mx-auto mt-2 max-w-sm text-sm text-muted-foreground">
          Connect your wallet to view your challenge history.
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex flex-wrap gap-1.5" role="group" aria-label="Outcome filter">
          {FILTER_OPTIONS.map(({ value, label }) => (
            <Button
              key={value}
              variant={outcomeFilter === value ? 'default' : 'outline'}
              size="sm"
              onClick={() => setOutcomeFilter(value)}
            >
              {label}
            </Button>
          ))}
        </div>

        <div className="flex items-center gap-2 text-sm" role="group" aria-label="Date range">
          <label className="sr-only" htmlFor="history-date-from">From</label>
          <input
            id="history-date-from"
            type="date"
            value={dateFrom}
            onChange={(e) => setDateFrom(e.target.value)}
            className="h-8 rounded-md border border-border bg-card px-2 text-xs text-foreground"
          />
          <span className="text-xs text-muted-foreground" aria-hidden>–</span>
          <label className="sr-only" htmlFor="history-date-to">To</label>
          <input
            id="history-date-to"
            type="date"
            value={dateTo}
            onChange={(e) => setDateTo(e.target.value)}
            className="h-8 rounded-md border border-border bg-card px-2 text-xs text-foreground"
          />
          {(dateFrom || dateTo) && (
            <Button
              variant="ghost"
              size="xs"
              onClick={() => { setDateFrom(''); setDateTo('') }}
            >
              Clear
            </Button>
          )}
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          Loading history…
        </div>
      ) : filtered.length === 0 ? (
        <div className="rounded-lg border border-dashed bg-muted/40 px-6 py-12 text-center">
          <p className="font-heading text-2xl">No history yet</p>
          <p className="mx-auto mt-2 max-w-sm text-sm text-muted-foreground">
            {entries.length === 0
              ? 'Settle a wager to see it appear here.'
              : 'No entries match the current filters.'}
          </p>
        </div>
      ) : (
        <ul className="grid list-none gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {filtered.map((entry) => {
            const labels = entry.match ? matchLabels(entry.match) : null
            const OutcomeIcon = outcomeIcon(entry.outcome)
            const awaitingOpponent =
              isPlaceholderAddress(entry.opponent) || entry.opponent.length === 0
            const scoreLine = labels?.scoreLine

            return (
              <li key={entry.wager.pubkey}>
                <Card
                  className="cursor-pointer overflow-hidden transition-colors hover:bg-muted/20"
                  onClick={() => navigate(`/my-wagers/${entry.wager.pubkey}`)}
                >
                  <CardContent className="space-y-3 px-4 py-4">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0 space-y-0.5">
                        <p className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
                          <Swords className="size-3 shrink-0" aria-hidden />
                          {entry.isMaker ? 'You challenged' : 'You accepted'}
                        </p>
                        {labels && (
                          <p className="text-[11px] text-muted-foreground">
                            {labels.league}
                            {entry.date > 0 && (
                              <>
                                <span className="mx-1" aria-hidden>·</span>
                                {formatDate(entry.date)}
                              </>
                            )}
                          </p>
                        )}
                      </div>
                      <span
                        className={cn(
                          'inline-flex h-6 items-center gap-1 rounded-full border px-2.5 text-xs font-medium shrink-0',
                          outcomeBadgeClass(entry.outcome),
                        )}
                      >
                        <OutcomeIcon className="size-3" aria-hidden />
                        {outcomeLabel(entry.outcome)}
                      </span>
                    </div>

                    {labels ? (
                      <DuelFrame
                        home={labels.homeTeam}
                        away={labels.awayTeam}
                        size="dense"
                        layout="inline"
                      />
                    ) : (
                      <p className="font-heading text-base">
                        Match {entry.wager.match_id}
                      </p>
                    )}

                    {scoreLine && (
                      <p className="text-center text-sm font-semibold tabular-nums">
                        {scoreLine}
                      </p>
                    )}

                    <div className="grid gap-2 rounded-md bg-muted/40 px-3 py-2.5 text-xs">
                      <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
                        <span className="text-muted-foreground">
                          Your pick{' '}
                          <span className="font-medium text-foreground">{entry.backedLabel}</span>
                        </span>
                        <span className="tabular-nums text-foreground">
                          <span className="font-medium">{entry.stakeFmt}</span>{' '}
                          <span className="text-muted-foreground">{stablecoin}</span>
                        </span>
                      </div>
                      {entry.outcome === 'won' && (
                        <div className="flex items-center justify-between border-t border-border/50 pt-2">
                          <span className="text-muted-foreground">Payout</span>
                          <span className="tabular-nums font-semibold text-primary">
                            {formatStakeBaseUnits(entry.wager.stake * 2)} {stablecoin}
                          </span>
                        </div>
                      )}
                    </div>

                    {!awaitingOpponent ? (
                      <p className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
                        <User className="size-3 shrink-0" aria-hidden />
                        Opponent{' '}
                        <span className="font-mono">{truncateAddress(entry.opponent)}</span>
                      </p>
                    ) : null}
                  </CardContent>
                </Card>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
