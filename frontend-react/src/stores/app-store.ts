import { create } from 'zustand'

import { MatchlockApi } from '@/lib/api'
import { tryLoadConfig, type AppConfig } from '@/lib/config'

type AppStore = {
  config: AppConfig
  configError: string | null
  api: MatchlockApi
}

const configResult = tryLoadConfig()

const fallbackConfig: AppConfig = {
  rpcUrl: 'https://api.devnet.solana.com',
  programId: 'VgsUt4Fjn6jqrqP7EuqvWJM3NqYufA2haNrP9fGGaYv',
  backendUrl: 'http://localhost:8080',
  cluster: 'devnet',
  usdcMint: 'ELWTKspHKCnCfCiCiqYw1EDH77k8VCP74dK9qytG2Ujh',
}

const config = configResult.ok ? configResult.config : fallbackConfig

export const useAppStore = create<AppStore>(() => ({
  config,
  configError: configResult.ok ? null : configResult.error,
  api: new MatchlockApi(config),
}))