import { useAppStore } from '@/stores/app-store'

export function useApi() {
  return useAppStore((state) => state.api)
}

export function useConfig() {
  return useAppStore((state) => state.config)
}