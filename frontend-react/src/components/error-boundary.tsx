import { Component, type ErrorInfo, type ReactNode } from 'react'

import { Button } from '@/components/ui/button'

type ErrorBoundaryProps = {
  children: ReactNode
}

type ErrorBoundaryState = {
  error: Error | null
}

export class ErrorBoundary extends Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  state: ErrorBoundaryState = { error: null }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Matchlock render error', error, info.componentStack)
  }

  render() {
    if (this.state.error) {
      return (
        <div className="flex min-h-svh items-center justify-center bg-background p-6">
          <div className="max-w-md rounded-lg border bg-card p-6 shadow-sahara">
            <h1 className="font-heading text-2xl">Something went wrong</h1>
            <p className="mt-2 text-sm text-muted-foreground">
              The app hit an unexpected error while loading.
            </p>
            <pre className="mt-4 max-h-48 overflow-auto rounded-md bg-muted p-3 text-xs whitespace-pre-wrap">
              {this.state.error.message}
            </pre>
            <Button
              className="mt-4"
              onClick={() => window.location.reload()}
            >
              Reload page
            </Button>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}