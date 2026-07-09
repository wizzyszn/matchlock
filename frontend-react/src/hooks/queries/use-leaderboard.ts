import { useQuery } from '@tanstack/react-query'

import { useApi } from '@/hooks/use-api'

export function useLeaderboardQuery(limit = 20) {
  const api = useApi()

  return useQuery({
    queryKey: ['leaderboard', limit],
    queryFn: () => api.getLeaderboard(limit),
    refetchInterval: 60_000,
  })
}

export function useMyLeaderboardRankQuery() {
  const api = useApi()

  return useQuery({
    queryKey: ['leaderboard', 'me'],
    queryFn: () => api.getMyLeaderboardRank(),
    retry: false,
  })
}

export function useLeaderboardStatsQuery() {
  const api = useApi()

  return useQuery({
    queryKey: ['leaderboard', 'stats'],
    queryFn: () => api.getLeaderboardStats(),
    refetchInterval: 60_000,
  })
}