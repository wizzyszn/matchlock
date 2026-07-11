import type { WagerStatus } from '@/lib/api'

export type WagerListParams = {
  match_id?: string
  status?: WagerStatus
  wallet?: string
}

export type WagerHistoryListParams = {
  wallet: string
  settlement_status?: 'settled' | 'unsettled'
  outcome?: 'won' | 'lost' | 'void'
  from?: number
  to?: number
  offset?: number
  limit?: number
}

export const queryKeys = {
  health: ['health'] as const,
  matches: {
    all: ['matches'] as const,
    detail: (id: string) => ['matches', id] as const,
  },
  wagers: {
    list: (params: WagerListParams = {}) => ['wagers', params] as const,
    history: (params: WagerHistoryListParams) => ['wagers', 'history', params] as const,
    detail: (pubkey: string) => ['wagers', pubkey] as const,
    settlement: (pubkey: string) => ['wagers', pubkey, 'settlement'] as const,
  },
  tokenBalance: (owner: string, mint: string) =>
    ['tokenBalance', owner, mint] as const,
  auth: {
    session: ['auth', 'session'] as const,
    walletBinding: (pubkey: string) => ['auth', 'wallet-binding', pubkey] as const,
    invites: ['invites'] as const,
    invite: (id: string) => ['invites', id] as const,
  },
}
