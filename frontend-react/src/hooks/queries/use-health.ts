import { useQuery } from '@tanstack/react-query'

import { useApi } from '@/hooks/use-api'
import { queryKeys } from '@/lib/query-keys'

export function useHealthQuery() {
  const api = useApi()

  return useQuery({
    queryKey: queryKeys.health,
    queryFn: () => api.getHealthz(),
    refetchInterval: 30_000,
  })
}