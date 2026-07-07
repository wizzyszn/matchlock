import {
  ConnectionProvider,
  WalletProvider as SolanaWalletProvider,
} from '@solana/wallet-adapter-react'
import { PhantomWalletAdapter } from '@solana/wallet-adapter-phantom'
import { SolflareWalletAdapter } from '@solana/wallet-adapter-solflare'
import { useMemo, type ReactNode } from 'react'

import { useAppStore } from '@/stores/app-store'

type WalletProviderProps = {
  children: ReactNode
}

export function WalletProvider({ children }: WalletProviderProps) {
  const rpcUrl = useAppStore((state) => state.config.rpcUrl)

  const wallets = useMemo(
    () => [new PhantomWalletAdapter(), new SolflareWalletAdapter()],
    [],
  )

  const endpoint = useMemo(() => rpcUrl, [rpcUrl])

  return (
    <ConnectionProvider endpoint={endpoint}>
      <SolanaWalletProvider wallets={wallets} autoConnect>
        {children}
      </SolanaWalletProvider>
    </ConnectionProvider>
  )
}