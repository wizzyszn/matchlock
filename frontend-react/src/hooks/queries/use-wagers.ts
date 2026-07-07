import { useQuery } from '@tanstack/react-query'

import { ApiClientError } from '@/lib/api'
import { useApi } from '@/hooks/use-api'
import { queryKeys, type WagerListParams } from '@/lib/query-keys'

export function useWagersQuery(params: WagerListParams = {}) {
  const api = useApi()

  return useQuery({
    queryKey: queryKeys.wagers.list(params),
    queryFn: () => api.listWagers(params),
    refetchInterval: 10_000,
  })
}

export function useWagerSettlementQuery(pubkey: string | undefined) {
  const api = useApi()

  return useQuery({
    queryKey: queryKeys.wagers.settlement(pubkey ?? ''),
    queryFn: () => api.getWagerSettlement(pubkey!),
    enabled: Boolean(pubkey),
    refetchInterval: (query) => {
      const state = query.state.data?.state
      if (!state || state === 'settled' || state === 'not_applicable') {
        return false
      }
      return 5_000
    },
  })
}

export function useWagerQuery(pubkey: string | undefined) {
  const api = useApi()

  return useQuery({
    queryKey: queryKeys.wagers.detail(pubkey ?? ''),
    queryFn: async () => {
      try {
        return await api.getWager(pubkey!)
      } catch (error) {
        if (error instanceof ApiClientError && error.status === 404) {
          return null
        }
        throw error
      }
    },
    enabled: Boolean(pubkey),
    refetchInterval: (query) => {
      const wager = query.state.data
      if (wager === null) return false
      if (wager?.status === 'matched') return 5_000
      return false
    },
  })
}