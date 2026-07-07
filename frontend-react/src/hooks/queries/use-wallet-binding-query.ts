import { useQuery } from '@tanstack/react-query'

import { useApi } from '@/hooks/use-api'
import { queryKeys } from '@/lib/query-keys'

export function useWalletBindingQuery(pubkey: string | null) {
  const api = useApi()

  return useQuery({
    queryKey: queryKeys.auth.walletBinding(pubkey ?? ''),
    queryFn: () => api.checkWalletBinding(pubkey!),
    enabled: Boolean(pubkey),
    staleTime: 30_000,
  })
}