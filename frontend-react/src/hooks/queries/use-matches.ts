import { useQuery } from '@tanstack/react-query'

import { useApi } from '@/hooks/use-api'
import { queryKeys } from '@/lib/query-keys'

export function useMatchesQuery() {
  const api = useApi()

  return useQuery({
    queryKey: queryKeys.matches.all,
    queryFn: () => api.listMatches(),
    refetchInterval: 30_000,
  })
}

export function useMatchQuery(matchId: string | undefined) {
  const api = useApi()

  return useQuery({
    queryKey: queryKeys.matches.detail(matchId ?? ''),
    queryFn: () => api.getMatch(matchId!),
    enabled: Boolean(matchId),
    refetchInterval: (query) => {
      const match = query.state.data
      if (!match || match.is_final) return false
      return 15_000
    },
  })
}