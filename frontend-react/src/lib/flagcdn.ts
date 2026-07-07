/**
 * Flagpedia CDN — https://flagpedia.net/download/api
 * Embed pattern: https://flagcdn.com/{size}/{iso}.{format}
 */

export const FLAGCDN_BASE = 'https://flagcdn.com'

/** Waving icons (4:3). Good for inline table rows. */
export type FlagIconSize =
  | '16x12'
  | '20x15'
  | '24x18'
  | '32x24'
  | '40x30'
  | '48x36'
  | '64x48'
  | '80x60'
  | '96x72'

/** Fixed width; height varies by flag aspect. */
export type FlagWidthSize = 'w20' | 'w40' | 'w80' | 'w160'

/** Fixed height; width varies by flag aspect. */
export type FlagHeightSize = 'h20' | 'h24' | 'h40'

export type FlagCdnSize = FlagIconSize | FlagWidthSize | FlagHeightSize

export type FlagFormat = 'png' | 'webp'

export function flagCdnUrl(
  isoCode: string,
  size: FlagCdnSize = '24x18',
  format: FlagFormat = 'png',
): string {
  const code = isoCode.trim().toLowerCase()
  return `${FLAGCDN_BASE}/${size}/${code}.${format}`
}

export function flagCdnSrcSet(
  isoCode: string,
  size1x: FlagCdnSize,
  size2x: FlagCdnSize,
  format: FlagFormat = 'png',
): string {
  return `${flagCdnUrl(isoCode, size1x, format)} 1x, ${flagCdnUrl(isoCode, size2x, format)} 2x`
}

export const FLAG_PRESETS = {
  sm: {
    src: '24x18' as const,
    src2x: '48x36' as const,
    className: 'h-[18px] w-6',
  },
  md: {
    src: '32x24' as const,
    src2x: '64x48' as const,
    className: 'h-6 w-8',
  },
  lg: {
    src: '40x30' as const,
    src2x: '80x60' as const,
    className: 'h-9 w-12',
  },
  xl: {
    src: '48x36' as const,
    src2x: '96x72' as const,
    className: 'h-11 w-[3.65rem]',
  },
} as const

export type FlagPreset = keyof typeof FLAG_PRESETS
