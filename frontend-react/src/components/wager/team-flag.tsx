import { useState } from 'react'

import {
  FLAG_PRESETS,
  flagCdnSrcSet,
  flagCdnUrl,
  type FlagPreset,
} from '@/lib/flagcdn'
import { teamNameToFlagCode } from '@/lib/team-flag-codes'
import { cn } from '@/lib/utils'

export type TeamFlagProps = {
  name: string
  /** Override ISO code (Flagpedia / flagcdn.com). */
  code?: string | null
  size?: FlagPreset
  className?: string
}

function InitialBadge({ name, size }: { name: string; size: FlagPreset }) {
  const initial = name.trim().charAt(0).toUpperCase() || '?'
  const dim = size === 'xl' ? 'size-11 text-sm' : size === 'lg' ? 'size-9 text-xs' : 'size-5 text-[10px]'

  return (
    <span
      className={cn(
        'inline-flex shrink-0 items-center justify-center rounded-full border border-border bg-muted font-semibold text-muted-foreground',
        dim,
      )}
      aria-hidden
    >
      {initial}
    </span>
  )
}

export function TeamFlag({
  name,
  code,
  size = 'sm',
  className,
}: TeamFlagProps) {
  const [failed, setFailed] = useState(false)
  const resolved = (code ?? teamNameToFlagCode(name))?.toLowerCase() ?? null
  const preset = FLAG_PRESETS[size]

  if (!resolved || failed) {
    return (
      <span className={cn('inline-flex shrink-0', className)}>
        <InitialBadge name={name} size={size} />
      </span>
    )
  }

  return (
    <img
      src={flagCdnUrl(resolved, preset.src)}
      srcSet={flagCdnSrcSet(resolved, preset.src, preset.src2x)}
      sizes={preset.className.includes('w-') ? undefined : '1.5rem'}
      alt=""
      aria-hidden
      loading="lazy"
      decoding="async"
      className={cn(
        'inline-block shrink-0 rounded-sm border border-border/80 object-cover shadow-sm',
        preset.className,
        className,
      )}
      onError={() => setFailed(true)}
    />
  )
}