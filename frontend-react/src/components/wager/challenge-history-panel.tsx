import { useWallet } from '@solana/wallet-adapter-react'
import {
  ChevronLeft,
  ChevronRight,
  Loader2,
  Swords,
  User,
} from 'lucide-react'
import { useCallback, useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useWagerHistoryQuery } from '@/hooks/queries/use-wagers'
import { useStablecoinLabel } from '@/hooks/use-stablecoin-label'
import type { WagerHistoryEntry } from '@/lib/api'
import { formatStakeBaseUnits, truncateAddress } from '@/lib/format'
import { matchLabels } from '@/lib/match-display'
import { cn } from '@/lib/utils'
import { isPlaceholderAddress } from '@/lib/accounts'

const PAGE_SIZE = 25

type StatusFilter = 'all' | 'settled' | 'unsettled'
type OutcomeFilter = 'all' | 'won' | 'lost' | 'void'

const STATUS_OPTIONS: { value: StatusFilter; label: string }[] = [
  { value: 'all', label: 'All bets' },
  { value: 'settled', label: 'Settled' },
  { value: 'unsettled', label: 'Unsettled' },
]

const OUTCOME_OPTIONS: { value: OutcomeFilter; label: string }[] = [
  { value: 'all', label: 'All outcomes' },
  { value: 'won', label: 'Won' },
  { value: 'lost', label: 'Lost' },
  { value: 'void', label: 'Void' },
]

type DatePreset = 'all' | 'today' | '7d' | '30d' | 'custom'

const DATE_PRESETS: { value: DatePreset; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'today', label: 'Today' },
  { value: '7d', label: '7d' },
  { value: '30d', label: '30d' },
  { value: 'custom', label: 'Custom' },
]

function outcomeLabel(outcome: NonNullable<WagerHistoryEntry['outcome']>) {
  switch (outcome) {
    case 'won':
      return 'Won'
    case 'lost':
      return 'Lost'
    case 'void':
      return 'Void'
  }
}

function outcomeBadgeClass(outcome: NonNullable<WagerHistoryEntry['outcome']>) {
  switch (outcome) {
    case 'won':
      return 'border-status-settled/25 bg-status-settled-bg text-status-settled'
    case 'lost':
      return 'border-status-cancelled/25 bg-status-cancelled-bg text-status-cancelled'
    case 'void':
      return 'border-muted-foreground/25 bg-muted/40 text-muted-foreground'
  }
}

function statusBadgeClass(status: WagerHistoryEntry['wager']['status']) {
  switch (status) {
    case 'matched':
      return 'border-status-matched/25 bg-status-matched-bg text-status-matched'
    case 'open':
      return 'border-status-open/25 bg-status-open-bg text-status-open'
    default:
      return 'border-muted-foreground/25 bg-muted/40 text-muted-foreground'
  }
}

function statusLabel(status: WagerHistoryEntry['wager']['status']) {
  switch (status) {
    case 'open':
      return 'Open'
    case 'matched':
      return 'Matched'
    case 'settled':
      return 'Settled'
    case 'cancelled':
      return 'Cancelled'
  }
}

function formatShortDate(ms: number): string {
  if (ms <= 0) return ''
  return new Intl.DateTimeFormat('en-GB', {
    day: '2-digit',
    month: 'short',
  }).format(new Date(ms))
}

function dateInputToStartMs(value: string): number | null {
  if (!value) return null
  const [year, month, day] = value.split('-').map(Number)
  if (!year || !month || !day) return null
  return new Date(year, month - 1, day).getTime()
}

type FilterSelectProps<TValue extends string> = {
  label: string
  value: TValue
  options: { value: TValue; label: string }[]
  onChange: (value: TValue) => void
}

function FilterSelect<TValue extends string>({
  label,
  value,
  options,
  onChange,
}: FilterSelectProps<TValue>) {
  return (
    <label className="grid gap-1.5 text-sm">
      <span className="text-xs font-medium text-muted-foreground">{label}</span>
      <select
        value={value}
        onChange={(event) => onChange(event.target.value as TValue)}
        className="flex h-11 w-full min-w-0 rounded-md border border-input bg-background px-3 py-2 text-sm shadow-xs outline-none transition-colors focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30"
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  )
}

export function ChallengeHistoryPanel() {
  const { publicKey } = useWallet()
  const walletAddress = publicKey?.toBase58()
  const navigate = useNavigate()
  const stablecoin = useStablecoinLabel()

  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [outcomeFilter, setOutcomeFilter] = useState<OutcomeFilter>('all')
  const [datePreset, setDatePreset] = useState<DatePreset>('all')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [page, setPage] = useState(0)

  const fromMs = dateInputToStartMs(dateFrom)
  const toMs = dateInputToStartMs(dateTo)
  const offset = page * PAGE_SIZE

  const { data: pageData, isLoading } = useWagerHistoryQuery(
    walletAddress
      ? {
          wallet: walletAddress,
          settlement_status:
            statusFilter === 'all' ? undefined : statusFilter,
          outcome: outcomeFilter === 'all' ? undefined : outcomeFilter,
          from: fromMs ?? undefined,
          to: toMs != null ? toMs + 86_399_999 : undefined,
          offset,
          limit: PAGE_SIZE,
        }
      : undefined,
  )

  const entries = pageData?.entries ?? []
  const total = pageData?.total ?? 0
  const hasMore = pageData?.has_more ?? false
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const currentPage = page + 1
  const hasActiveFilters =
    statusFilter !== 'all' ||
    outcomeFilter !== 'all' ||
    datePreset !== 'all'

  const handlePreset = useCallback((preset: DatePreset) => {
    setDatePreset(preset)
    setPage(0)
    const today = new Date()
    const fmt = (d: Date) => {
      const y = d.getFullYear()
      const m = String(d.getMonth() + 1).padStart(2, '0')
      const day = String(d.getDate()).padStart(2, '0')
      return `${y}-${m}-${day}`
    }
    switch (preset) {
      case 'all':
        setDateFrom('')
        setDateTo('')
        break
      case 'today': {
        const d = fmt(today)
        setDateFrom(d)
        setDateTo(d)
        break
      }
      case '7d': {
        const from = new Date(today)
        from.setDate(from.getDate() - 7)
        setDateFrom(fmt(from))
        setDateTo(fmt(today))
        break
      }
      case '30d': {
        const from = new Date(today)
        from.setDate(from.getDate() - 30)
        setDateFrom(fmt(from))
        setDateTo(fmt(today))
        break
      }
      case 'custom':
        break
    }
  }, [])

  const handleDateFromChange = useCallback(
    (value: string) => {
      setDateFrom(value)
      setDatePreset('custom')
      setPage(0)
    },
    [],
  )

  const handleDateToChange = useCallback(
    (value: string) => {
      setDateTo(value)
      setDatePreset('custom')
      setPage(0)
    },
    [],
  )

  const handleStatusChange = useCallback((value: StatusFilter) => {
    setStatusFilter(value)
    setPage(0)
  }, [])

  const handleOutcomeChange = useCallback((value: OutcomeFilter) => {
    setOutcomeFilter(value)
    setPage(0)
  }, [])

  const resetFilters = useCallback(() => {
    setStatusFilter('all')
    setOutcomeFilter('all')
    setDatePreset('all')
    setDateFrom('')
    setDateTo('')
    setPage(0)
  }, [])

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
    <div className="space-y-5">
      {/* --- Filters --- */}
      <div className="space-y-3 rounded-lg border bg-muted/20 p-4">
        <div className="grid gap-3 lg:grid-cols-[minmax(0,12rem)_minmax(0,12rem)_1fr]">
          <FilterSelect
            label="Settlement"
            value={statusFilter}
            options={STATUS_OPTIONS}
            onChange={handleStatusChange}
          />

          <FilterSelect
            label="Result"
            value={outcomeFilter}
            options={OUTCOME_OPTIONS}
            onChange={handleOutcomeChange}
          />

          <div className="grid gap-1.5 text-sm">
            <span className="text-xs font-medium text-muted-foreground">
              Date range
            </span>
            <div
              className="flex flex-wrap items-center gap-1.5"
              role="group"
              aria-label="Date range presets"
            >
              {DATE_PRESETS.map((preset) => (
                <button
                  key={preset.value}
                  type="button"
                  onClick={() => handlePreset(preset.value)}
                  className={cn(
                    'inline-flex h-8 items-center rounded-md border px-3 text-xs font-medium transition-colors',
                    datePreset === preset.value
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'border-input bg-background text-muted-foreground hover:border-ring hover:text-foreground',
                  )}
                >
                  {preset.label}
                </button>
              ))}
            </div>
            {datePreset === 'custom' && (
              <div
                className="mt-1 flex gap-2 sm:items-center"
                role="group"
                aria-label="Custom date range"
              >
                <div className="grid flex-1 gap-1">
                  <label
                    className="sr-only"
                    htmlFor="history-date-from"
                  >
                    From
                  </label>
                  <Input
                    id="history-date-from"
                    type="date"
                    value={dateFrom}
                    max={dateTo || undefined}
                    onChange={(e) =>
                      handleDateFromChange(e.target.value)
                    }
                    className="h-9 text-xs"
                  />
                </div>
                <span className="mt-1.5 hidden self-start text-xs text-muted-foreground sm:inline">
                  →
                </span>
                <div className="grid flex-1 gap-1">
                  <label className="sr-only" htmlFor="history-date-to">
                    To
                  </label>
                  <Input
                    id="history-date-to"
                    type="date"
                    value={dateTo}
                    min={dateFrom || undefined}
                    onChange={(e) =>
                      handleDateToChange(e.target.value)
                    }
                    className="h-9 text-xs"
                  />
                </div>
              </div>
            )}
          </div>
        </div>

        <div className="flex items-center justify-between gap-3">
          <p className="text-sm text-muted-foreground">
            <span className="font-medium text-foreground">{total}</span>{' '}
            {total === 1 ? 'bet' : 'bets'}
            {hasActiveFilters && (
              <>
                <span className="mx-1" aria-hidden>
                  ·
                </span>
                {statusFilter !== 'all' && (
                  <span className="capitalize">{statusFilter}</span>
                )}
                {outcomeFilter !== 'all' && (
                  <>
                    {statusFilter !== 'all' && ' / '}
                    <span className="capitalize">{outcomeFilter}</span>
                  </>
                )}
              </>
            )}
          </p>

          {hasActiveFilters && (
            <Button
              variant="ghost"
              size="sm"
              className="h-8 text-xs"
              onClick={resetFilters}
            >
              Reset filters
            </Button>
          )}
        </div>
      </div>

      {/* --- Content --- */}
      {isLoading ? (
        <div className="flex items-center justify-center gap-2 py-16 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          Loading history…
        </div>
      ) : entries.length === 0 ? (
        <div className="rounded-lg border border-dashed bg-muted/40 px-6 py-12 text-center">
          <p className="font-heading text-2xl">No bets found</p>
          <p className="mx-auto mt-2 max-w-sm text-sm text-muted-foreground">
            {!hasActiveFilters
              ? 'Create or accept a wager to build your history.'
              : 'No bets match the current filters.'}
          </p>
        </div>
      ) : (
        <>
          {/* --- Table --- */}
          <div className="overflow-x-auto rounded-lg border">
            <table className="w-full table-auto border-collapse text-sm">
              <thead>
                <tr className="border-b bg-muted/30 text-xs font-medium text-muted-foreground">
                  <th className="px-3 py-3 text-left font-medium">Match</th>
                  <th className="px-3 py-3 text-left font-medium max-sm:hidden">
                    Pick
                  </th>
                  <th className="px-3 py-3 text-right font-medium">Stake</th>
                  <th className="px-3 py-3 text-right font-medium max-sm:hidden">
                    Result
                  </th>
                  <th className="px-3 py-3 text-center font-medium">Status</th>
                </tr>
              </thead>
              <tbody>
                {entries.map((entry) => {
                  const labels = entry.match
                    ? matchLabels(entry.match)
                    : null
                  const eventTime =
                    entry.event_time ?? entry.match?.start_time ?? 0
                  const homeName = labels?.homeTeam ?? 'Home'
                  const awayName = labels?.awayTeam ?? 'Away'
                  const scoreLine = labels?.scoreLine
                  const stakeFmt = formatStakeBaseUnits(entry.wager.stake)
                  const backedLabel = entry.match
                    ? (() => {
                        const s = entry.backed_side
                        const lbl = matchLabels(entry.match)
                        switch (s) {
                          case 'home':
                            return lbl.homeTeam
                          case 'away':
                            return lbl.awayTeam
                          case 'draw':
                            return 'Draw'
                          case 'unset':
                            return '—'
                        }
                      })()
                    : entry.backed_side

                  const payoutAmount =
                    entry.outcome === 'won'
                      ? entry.wager.stake * 2
                      : entry.outcome === 'void'
                        ? entry.wager.stake
                        : null
                  const payoutFmt = payoutAmount != null
                    ? formatStakeBaseUnits(payoutAmount)
                    : null

                  const awaitingOpponent =
                    !entry.opponent ||
                    isPlaceholderAddress(entry.opponent) ||
                    entry.opponent.length === 0

                  return (
                    <tr
                      key={entry.wager.pubkey}
                      onClick={() =>
                        navigate(`/my-wagers/${entry.wager.pubkey}`)
                      }
                      className="cursor-pointer border-b border-border/50 transition-colors last:border-b-0 hover:bg-muted/20"
                    >
                      {/* Match */}
                      <td className="px-3 py-3">
                        <div className="flex flex-col gap-0.5">
                          <div className="flex items-center gap-1.5">
                            <Swords className="size-3 shrink-0 text-muted-foreground" />
                            <span className="truncate font-medium">
                              {homeName}
                            </span>
                            {scoreLine ? (
                              <span className="shrink-0 text-xs font-semibold tabular-nums text-foreground">
                                {scoreLine}
                              </span>
                            ) : (
                              <span className="shrink-0 text-[10px] font-medium tracking-widest text-muted-foreground uppercase">
                                vs
                              </span>
                            )}
                            <span className="truncate font-medium">
                              {awayName}
                            </span>
                          </div>
                          <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
                            {entry.is_maker
                              ? 'You challenged'
                              : 'You accepted'}
                            {eventTime > 0 && (
                              <>
                                <span aria-hidden>·</span>
                                <time dateTime={new Date(eventTime).toISOString()}>
                                  {formatShortDate(eventTime)}
                                </time>
                              </>
                            )}
                            {!awaitingOpponent && (
                              <>
                                <span aria-hidden>·</span>
                                <span className="flex items-center gap-1">
                                  <User className="size-2.5" />
                                  {truncateAddress(entry.opponent!)}
                                </span>
                              </>
                            )}
                          </div>
                        </div>
                      </td>

                      {/* Pick */}
                      <td className="px-3 py-3 max-sm:hidden">
                        <span className="text-xs font-medium">
                          {backedLabel}
                        </span>
                      </td>

                      {/* Stake */}
                      <td className="px-3 py-3 text-right">
                        <span className="whitespace-nowrap font-medium tabular-nums">
                          {stakeFmt}
                        </span>{' '}
                        <span className="text-xs text-muted-foreground">
                          {stablecoin}
                        </span>
                      </td>

                      {/* Result */}
                      <td className="px-3 py-3 text-right max-sm:hidden">
                        {payoutFmt != null ? (
                          <span
                            className={cn(
                              'whitespace-nowrap font-semibold tabular-nums',
                              entry.outcome === 'won'
                                ? 'text-primary'
                                : 'text-foreground',
                            )}
                          >
                            {entry.outcome === 'won' ? '+' : ''}
                            {payoutFmt} {stablecoin}
                          </span>
                        ) : entry.settlement_status === 'unsettled' ? (
                          <span className="text-xs text-muted-foreground">
                            Pending
                          </span>
                        ) : (
                          <span className="text-xs text-muted-foreground">
                            —
                          </span>
                        )}
                      </td>

                      {/* Status */}
                      <td className="px-3 py-3 text-center">
                        {entry.outcome ? (
                          <span
                            className={cn(
                              'inline-flex h-5 items-center gap-1 rounded-full border px-2 text-[11px] font-medium whitespace-nowrap',
                              outcomeBadgeClass(entry.outcome),
                            )}
                          >
                            {outcomeLabel(entry.outcome)}
                          </span>
                        ) : (
                          <span
                            className={cn(
                              'inline-flex h-5 items-center gap-1 rounded-full border px-2 text-[11px] font-medium whitespace-nowrap',
                              statusBadgeClass(entry.wager.status),
                            )}
                          >
                            {statusLabel(entry.wager.status)}
                          </span>
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>

          {/* --- Pagination --- */}
          <div className="flex items-center justify-between gap-3">
            <p className="text-xs text-muted-foreground">
              Page {currentPage} of {totalPages}
              {total > 0 && (
                <>
                  <span className="mx-1" aria-hidden>
                    ·
                  </span>
                  {total} total
                </>
              )}
            </p>

            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page === 0}
                onClick={() => setPage((p) => Math.max(0, p - 1))}
                className="h-8 gap-1 px-2.5 text-xs"
              >
                <ChevronLeft className="size-3.5" />
                <span className="max-sm:sr-only">Previous</span>
              </Button>

              <Button
                variant="outline"
                size="sm"
                disabled={!hasMore}
                onClick={() => setPage((p) => p + 1)}
                className="h-8 gap-1 px-2.5 text-xs"
              >
                <span className="max-sm:sr-only">Next</span>
                <ChevronRight className="size-3.5" />
              </Button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
