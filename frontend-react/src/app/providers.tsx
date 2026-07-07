import { QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import { ErrorBoundary } from '@/components/error-boundary'
import { WalletProvider } from '@/components/wallet/WalletProvider'
import { queryClient } from '@/lib/query-client'
import { useAppStore } from '@/stores/app-store'
import { useMatchStream } from '@/hooks/use-match-stream'

function ConfigErrorScreen({ message }: { message: string }) {
  return (
    <div className="flex min-h-svh items-center justify-center bg-background p-6">
      <div className="max-w-md rounded-lg border bg-card p-6 shadow-sahara">
        <h1 className="font-heading text-2xl">Configuration error</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Copy <code className="text-xs">frontend-react/.env</code> and restart
          the dev server.
        </p>
        <pre className="mt-4 overflow-auto rounded-md bg-muted p-3 text-xs whitespace-pre-wrap">
          {message}
        </pre>
      </div>
    </div>
  )
}

type AppProvidersProps = {
  children: ReactNode
}

function MatchStreamSubscriber() {
  useMatchStream()
  return null
}

export function AppProviders({ children }: AppProvidersProps) {
  const configError = useAppStore((state) => state.configError)

  if (configError) {
    return <ConfigErrorScreen message={configError} />
  }

  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        {/* Mount the SSE subscriber inside the provider so it can access useQueryClient */}
        <MatchStreamSubscriber />
        <WalletProvider>{children}</WalletProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  )
}