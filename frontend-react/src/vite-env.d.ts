/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_SOLANA_RPC_URL: string
  readonly VITE_PROGRAM_ID: string
  readonly VITE_BACKEND_URL: string
  readonly VITE_CLUSTER: 'devnet' | 'mainnet-beta' | 'testnet' | 'localnet'
  readonly VITE_USDT_MINT: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}