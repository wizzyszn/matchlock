import { z } from 'zod'

const clusterSchema = z.enum(['devnet', 'mainnet-beta', 'testnet', 'localnet'])

const envSchema = z.object({
  VITE_SOLANA_RPC_URL: z.url(),
  VITE_PROGRAM_ID: z.string().min(32),
  VITE_BACKEND_URL: z.url(),
  VITE_CLUSTER: clusterSchema,
  VITE_USDT_MINT: z.string().min(32),
})

export type Cluster = z.infer<typeof clusterSchema>

export type AppConfig = {
  rpcUrl: string
  programId: string
  backendUrl: string
  cluster: Cluster
  usdtMint: string
}

export type ConfigResult =
  | { ok: true; config: AppConfig }
  | { ok: false; error: string }

export function loadConfig(): AppConfig {
  const result = tryLoadConfig()
  if (!result.ok) {
    throw new Error(result.error)
  }
  return result.config
}

export function tryLoadConfig(): ConfigResult {
  const parsed = envSchema.safeParse(import.meta.env)

  if (!parsed.success) {
    const details = parsed.error.issues
      .map((issue) => `${issue.path.join('.')}: ${issue.message}`)
      .join('; ')
    return { ok: false, error: `Invalid frontend environment: ${details}` }
  }

  const env = parsed.data

  return {
    ok: true,
    config: {
      rpcUrl: env.VITE_SOLANA_RPC_URL,
      programId: env.VITE_PROGRAM_ID,
      backendUrl: env.VITE_BACKEND_URL,
      cluster: env.VITE_CLUSTER,
      usdtMint: env.VITE_USDT_MINT,
    },
  }
}

export function clusterLabel(cluster: Cluster): string {
  switch (cluster) {
    case 'devnet':
      return 'Devnet'
    case 'mainnet-beta':
      return 'Mainnet'
    case 'testnet':
      return 'Testnet'
    case 'localnet':
      return 'Localnet'
  }
}

export function explorerClusterParam(cluster: Cluster): string {
  return cluster === 'mainnet-beta' ? '' : `?cluster=${cluster}`
}

/** User-facing stablecoin label — devnet uses TxLINE test USDT, not Circle USDC. */
export function stablecoinLabel(cluster: Cluster): string {
  return cluster === 'mainnet-beta' ? 'USDC' : 'USDT (devnet)'
}