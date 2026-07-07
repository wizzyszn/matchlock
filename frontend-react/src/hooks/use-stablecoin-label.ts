import { useConfig } from '@/hooks/use-api'
import { stablecoinLabel } from '@/lib/config'

export function useStablecoinLabel(): string {
  const config = useConfig()
  return stablecoinLabel(config.cluster)
}