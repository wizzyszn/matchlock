import { create } from 'zustand'

import type { Wager } from '@/lib/api'

type OptimisticEntry = {
  wager: Wager
  createdAt: number
  visibleUntil?: number
  expiresAt?: number
  hidden?: boolean
}

type OptimisticWagersStore = {
  wagers: Record<string, OptimisticEntry>
  upsert: (wager: Wager) => void
  markCancelled: (wager: Wager) => void
  remove: (pubkey: string) => void
  reconcile: (serverWagers: Wager[]) => void
  pruneExpired: () => void
}

const CREATE_TTL_MS = 120_000
const CANCEL_VISIBILITY_MS = 3_000
const CANCEL_SUPPRESSION_MS = 120_000

export const useOptimisticWagersStore = create<OptimisticWagersStore>((set) => ({
  wagers: {},
  upsert: (wager) =>
    set((state) => ({
      wagers: {
        ...state.wagers,
        [wager.pubkey]: {
          wager,
          createdAt: Date.now(),
          hidden: false,
        },
      },
    })),
  markCancelled: (wager) =>
    set((state) => {
      const existing = state.wagers[wager.pubkey]
      const now = Date.now()
      return {
        wagers: {
          ...state.wagers,
          [wager.pubkey]: {
            wager: { ...(existing?.wager ?? wager), ...wager, status: 'cancelled' },
            createdAt: existing?.createdAt ?? now,
            visibleUntil: now + CANCEL_VISIBILITY_MS,
            expiresAt: now + CANCEL_SUPPRESSION_MS,
            hidden: false,
          },
        },
      }
    }),
  remove: (pubkey) =>
    set((state) => {
      const next = { ...state.wagers }
      delete next[pubkey]
      return { wagers: next }
    }),
  reconcile: (serverWagers) =>
    set((state) => {
      const next = { ...state.wagers }
      const now = Date.now()
      const serverPubkeys = new Set(serverWagers.map((wager) => wager.pubkey))

      for (const [pubkey, entry] of Object.entries(next)) {
        if (entry.expiresAt && entry.expiresAt <= now) {
          delete next[pubkey]
          continue
        }
        if (entry.visibleUntil) {
          if (entry.visibleUntil <= now && !entry.hidden) {
            next[pubkey] = { ...entry, hidden: true }
          }
          continue
        }
        if (serverPubkeys.has(pubkey) || now - entry.createdAt > CREATE_TTL_MS) {
          delete next[pubkey]
        }
      }

      return { wagers: next }
    }),
  pruneExpired: () =>
    set((state) => {
      const next = { ...state.wagers }
      const now = Date.now()
      for (const [pubkey, entry] of Object.entries(next)) {
        if (entry.expiresAt && entry.expiresAt <= now) {
          delete next[pubkey]
          continue
        }
        if (entry.visibleUntil && entry.visibleUntil <= now && !entry.hidden) {
          next[pubkey] = { ...entry, hidden: true }
          continue
        }
        if (!entry.visibleUntil && now - entry.createdAt > CREATE_TTL_MS) {
          delete next[pubkey]
        }
      }
      return { wagers: next }
    }),
}))
