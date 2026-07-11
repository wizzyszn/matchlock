import { ChevronDown, Loader2, TrendingUp, Users, Trophy, Medal, Percent } from 'lucide-react'

import {
  useLeaderboardQuery,
  useMyLeaderboardRankQuery,
  useLeaderboardStatsQuery,
} from '@/hooks/queries/use-leaderboard'
import { formatUsdc } from '@/lib/format'
import { cn } from '@/lib/utils'

function RankIcon({ rank }: { rank: number }) {
  if (rank === 1) return <Trophy className="size-5 text-yellow-400" aria-hidden />
  if (rank === 2) return <Medal className="size-5 text-gray-400" aria-hidden />
  if (rank === 3) return <Medal className="size-5 text-amber-600" aria-hidden />
  return <span className="w-5 text-center text-sm font-semibold text-muted-foreground">{rank}</span>
}

function PnLBadge({ value }: { value: number }) {
  if (value === 0) return <span className="text-muted-foreground">—</span>
  return (
    <span className={cn(
      'tabular-nums font-medium',
      value > 0 ? 'text-status-matched' : 'text-destructive',
    )}>
      {value > 0 ? '+' : ''}{formatUsdc(value)}
    </span>
  )
}

const PAGE_SIZE = 20

export function LeaderboardPage() {
  const {
    data,
    isLoading,
    isError,
    error,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
  } = useLeaderboardQuery(PAGE_SIZE)
  const { data: myRank } = useMyLeaderboardRankQuery()
  const { data: stats } = useLeaderboardStatsQuery()

  const pages = data?.pages ?? []
  const entries = pages.flatMap((page) => page.entries)
  const total = pages[0]?.total ?? 0

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
        Loading leaderboard…
      </div>
    )
  }

  if (isError) {
    return (
      <p className="text-sm text-destructive">
        {error instanceof Error ? error.message : 'Failed to load leaderboard'}
      </p>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-heading text-3xl tracking-tight sm:text-4xl">Leaderboard</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Top predictors ranked by net PnL. Updated after each settlement.
        </p>
      </div>

      {stats ? (
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <div className="rounded-lg border bg-card p-4">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <TrendingUp className="size-4" aria-hidden />
              <span>Total Volume</span>
            </div>
            <p className="mt-1 font-heading text-2xl font-semibold tabular-nums">
              {formatUsdc(stats.total_volume)} <span className="text-xs font-normal text-muted-foreground">USDC</span>
            </p>
          </div>
          <div className="rounded-lg border bg-card p-4">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Trophy className="size-4" aria-hidden />
              <span>Total Wagers</span>
            </div>
            <p className="mt-1 font-heading text-2xl font-semibold tabular-nums">
              {stats.total_wagers}
            </p>
          </div>
          <div className="rounded-lg border bg-card p-4">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Users className="size-4" aria-hidden />
              <span>Active Users</span>
            </div>
            <p className="mt-1 font-heading text-2xl font-semibold tabular-nums">
              {stats.total_users}
            </p>
          </div>
          <div className="col-span-2 rounded-lg border bg-card p-4 sm:col-span-1 lg:col-span-1">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Percent className="size-4" aria-hidden />
              <span>Avg Win Rate</span>
            </div>
            <p className="mt-1 font-heading text-2xl font-semibold tabular-nums">
              {stats.avg_win_rate.toFixed(1)}%
            </p>
          </div>
        </div>
      ) : null}

      {myRank && 'rank' in myRank && myRank.rank != null ? (
        <div className="rounded-lg border border-primary/30 bg-primary/5 px-4 py-3">
          <p className="text-sm text-muted-foreground">Your rank</p>
          <p className="font-heading text-2xl font-bold">#{myRank.rank}</p>
          <div className="mt-1 flex gap-4 text-xs text-muted-foreground">
            <span>{myRank.total_wagers} wagers</span>
            <span>{myRank.wins}W / {myRank.losses}L</span>
            <PnLBadge value={myRank.net_pnl} />
          </div>
        </div>
      ) : null}

      {entries.length ? (
        <div className="overflow-hidden rounded-lg border bg-card shadow-sahara">
          <ul className="divide-y divide-border/60">
            {entries.map((entry) => (
              <li key={entry.user_id} className="flex items-center gap-3 px-4 py-3">
                <div className="flex w-8 shrink-0 items-center justify-center">
                  <RankIcon rank={entry.rank} />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium text-foreground">
                    {entry.display_name || entry.email.split('@')[0]}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {entry.total_wagers} wagers · {entry.wins}W / {entry.losses}L
                  </p>
                </div>
                <div className="text-right">
                  <PnLBadge value={entry.net_pnl} />
                  <p className="text-xs text-muted-foreground tabular-nums">
                    {entry.total_volume.toLocaleString()} · {entry.win_rate.toFixed(0)}%
                  </p>
                </div>
              </li>
            ))}
          </ul>
          {hasNextPage ? (
            <div className="border-t border-border/60 px-4 py-3">
              <button
                type="button"
                onClick={() => void fetchNextPage()}
                disabled={isFetchingNextPage}
                className="flex w-full items-center justify-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
              >
                {isFetchingNextPage ? (
                  <Loader2 className="size-4 animate-spin" aria-hidden />
                ) : (
                  <ChevronDown className="size-4" aria-hidden />
                )}
                Load more ({Math.max(total - entries.length, 0)} remaining)
              </button>
            </div>
          ) : null}
        </div>
      ) : (
        <div className="rounded-lg border border-dashed bg-muted/40 px-6 py-12 text-center">
          <Trophy className="mx-auto size-8 text-muted-foreground" aria-hidden />
          <p className="font-heading text-xl">No settled wagers yet</p>
          <p className="mx-auto mt-2 max-w-sm text-sm text-muted-foreground">
            The leaderboard updates automatically after each settlement on-chain.
          </p>
        </div>
      )}
    </div>
  )
}
