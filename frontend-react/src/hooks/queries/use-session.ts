import { useQuery } from '@tanstack/react-query'

import { ApiClientError } from '@/lib/api'
import { useApi } from '@/hooks/use-api'
import { queryKeys } from '@/lib/query-keys'

export function useSessionQuery() {
  const api = useApi()

  return useQuery({
    queryKey: queryKeys.auth.session,
    queryFn: async () => {
      try {
        return await api.getMe()
      } catch (error) {
        if (error instanceof ApiClientError && error.status === 401) {
          try {
            return await api.refreshSession()
          } catch {
            return null
          }
        }
        throw error
      }
    },
    retry: false,
    staleTime: 60_000,
  })
}