import { useAnchorWallet, useConnection } from '@solana/wallet-adapter-react'
import { PublicKey } from '@solana/web3.js'
import { useQuery } from '@tanstack/react-query'

import { useConfig } from '@/hooks/use-api'
import { getProgram } from '@/lib/anchor'

type ConfigAccount = { paused: boolean }

export function useContractPaused() {
  const { connection } = useConnection()
  const wallet = useAnchorWallet()
  const config = useConfig()

  return useQuery({
    queryKey: ['contractPaused', config.programId],
    queryFn: async () => {
      if (!wallet?.publicKey) return null
      const program = getProgram(connection, wallet)
      const [configPda] = PublicKey.findProgramAddressSync(
        [Buffer.from('config')],
        program.programId,
      )
      const account = await (program.account as unknown as {
        config: { fetch: (address: PublicKey) => Promise<ConfigAccount> }
      }).config.fetch(configPda)
      return account.paused
    },
    refetchInterval: 30_000,
    staleTime: 15_000,
  })
}
