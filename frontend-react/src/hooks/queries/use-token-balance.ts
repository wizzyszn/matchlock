import { useConnection, useWallet } from '@solana/wallet-adapter-react'
import { useQuery } from '@tanstack/react-query'
import { PublicKey } from '@solana/web3.js'
import { getAssociatedTokenAddressSync } from '@solana/spl-token'

import { useConfig } from '@/hooks/use-api'
import { queryKeys } from '@/lib/query-keys'

export function useTokenBalanceQuery() {
  const { connection } = useConnection()
  const { publicKey } = useWallet()
  const config = useConfig()

  const owner = publicKey?.toBase58() ?? ''
  const mint = config.usdtMint

  return useQuery({
    queryKey: queryKeys.tokenBalance(owner, mint),
    queryFn: async () => {
      const ownerKey = publicKey!
      const mintKey = new PublicKey(mint)
      const ata = getAssociatedTokenAddressSync(mintKey, ownerKey)

      try {
        const account = await connection.getTokenAccountBalance(ata)
        return BigInt(account.value.amount)
      } catch {
        return BigInt(0)
      }
    },
    enabled: Boolean(publicKey),
    refetchInterval: 20_000,
  })
}