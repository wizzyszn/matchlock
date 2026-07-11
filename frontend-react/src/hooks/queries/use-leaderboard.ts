import { useInfiniteQuery, useQuery } from '@tanstack/react-query'

import { useApi } from '@/hooks/use-api'

export function useLeaderboardQuery(limit = 20) {
  const api = useApi()

  return useInfiniteQuery({
    queryKey: ['leaderboard', limit],
    queryFn: ({ pageParam }) => api.getLeaderboard(pageParam, limit),
    initialPageParam: 0,
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? lastPage.offset + lastPage.limit : undefined,
    refetchInterval: 60_000,
  })
}

export function useMyLeaderboardRankQuery() {
  const api = useApi()

  return useQuery({
    queryKey: ['leaderboard', 'me'],
    queryFn: () => api.getMyLeaderboardRank(),
    retry: false,
    refetchInterval: 60_000,
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
