import { Badge } from '@/components/ui/badge'
import { clusterLabel } from '@/lib/config'
import { useAppStore } from '@/stores/app-store'

export function ClusterBadge() {
  const cluster = useAppStore((state) => state.config.cluster)

  return (
    <Badge variant="outline" className="font-mono text-[0.7rem] tracking-wide">
      {clusterLabel(cluster)}
    </Badge>
  )
}