import { useWallet } from '@solana/wallet-adapter-react'

import { useWalletBindingQuery } from '@/hooks/queries/use-wallet-binding-query'
import { useSessionQuery } from '@/hooks/queries/use-session'
import type { WalletBinding, WalletLink } from '@/lib/api'

export type WalletLinkStatus = {
  connected: boolean
  connectedPubkey: string | null
  linkedWallets: WalletLink[]
  primaryWallet: WalletLink | undefined
  linkedMatch: WalletLink | undefined
  binding: WalletBinding | undefined
  isLinkedToAccount: boolean
  ownedByOther: boolean
  needsLink: boolean
  hasLinkedWallet: boolean
  mismatch: boolean
  conflict: boolean
  canLink: boolean
  canTransact: boolean
}

export function useWalletLinkStatus(): WalletLinkStatus {
  const { connected, publicKey } = useWallet()
  const { data: session } = useSessionQuery()

  const connectedPubkey = publicKey?.toBase58() ?? null
  const linkedWallets = session?.wallets ?? []
  const primaryWallet = linkedWallets.find((w) => w.is_primary)
  const linkedMatch = connectedPubkey
    ? linkedWallets.find((w) => w.pubkey === connectedPubkey)
    : undefined

  const { data: binding } = useWalletBindingQuery(connectedPubkey)

  const ownedByOther = Boolean(binding?.owned_by_other)
  const isLinkedToAccount = Boolean(
    binding?.linked_to_you ?? (linkedMatch && !ownedByOther),
  )
  const needsLink = connected && !isLinkedToAccount && !ownedByOther
  const hasLinkedWallet = linkedWallets.length > 0
  const mismatch =
    connected &&
    Boolean(primaryWallet) &&
    connectedPubkey !== primaryWallet?.pubkey &&
    !isLinkedToAccount &&
    !ownedByOther
  const conflict = connected && ownedByOther
  const canLink = connected && binding?.status === 'unlinked'
  const canTransact = connected && isLinkedToAccount && !ownedByOther

  return {
    connected,
    connectedPubkey,
    linkedWallets,
    primaryWallet,
    linkedMatch,
    binding,
    isLinkedToAccount,
    ownedByOther,
    needsLink,
    hasLinkedWallet,
    mismatch,
    conflict,
    canLink,
    canTransact,
  }
}
