import { useEffect } from 'react'

import { useQueryClient } from '@tanstack/react-query'

import { useApi, useConfig } from '@/hooks/use-api'
import type { Match } from '@/lib/api'
import { queryKeys } from '@/lib/query-keys'

export function useMatchStream() {
  const config = useConfig()
  const queryClient = useQueryClient()
  const api = useApi()

  useEffect(() => {
    // Only subscribe to SSE if we have a valid backend URL
    if (!config.backendUrl) return

    const streamUrl = `${config.backendUrl.replace(/\/$/, '')}/matches/stream`
    
    // We use native EventSource, which handles automatic reconnection with exponential backoff on its own.
    const eventSource = new EventSource(streamUrl)

    eventSource.onmessage = (event) => {
      try {
        const matchUpdate = JSON.parse(event.data) as Match
        
        // 1. Update the detail view cache for this specific match.
        queryClient.setQueryData(
          queryKeys.matches.detail(matchUpdate.match_id),
          (oldData: Match | undefined) => {
            // Only update if we don't have old data or if the new data is newer (based on req/res ordering, we assume its newer)
            return {
              ...(oldData ?? {}),
              ...matchUpdate
            } as Match
          }
        )

        // 2. Splice the update into the list view cache if it corresponds
        queryClient.setQueryData(
          queryKeys.matches.all,
          (oldList: Match[] | undefined) => {
            if (!oldList) return oldList
            
            return oldList.map((m) => 
              m.match_id === matchUpdate.match_id ? { ...m, ...matchUpdate } : m
            )
          }
        )

      } catch (err) {
        console.warn('Failed to parse match stream event:', err)
      }
    }

    eventSource.onerror = (err) => {
      // EventSource automatically reconnects, we just log it as debug.
      console.debug('Match stream connection error:', err)
    }

    return () => {
      eventSource.close()
    }
  }, [config.backendUrl, queryClient, api])
}
