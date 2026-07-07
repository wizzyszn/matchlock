import '@/polyfills'

import { RouterProvider } from 'react-router-dom'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { AppProviders } from '@/app/providers'
import { router } from '@/app/router'

import './index.css'

const rootEl = document.getElementById('root')

if (!rootEl) {
  throw new Error('Root element #root not found')
}

function showBootError(el: HTMLElement, message: string) {
  el.innerHTML = `
    <div style="font-family: system-ui, sans-serif; padding: 2rem; max-width: 40rem;">
      <h1 style="font-size: 1.5rem; margin-bottom: 0.5rem;">Matchlock failed to start</h1>
      <p style="color: #6b5f56; margin-bottom: 1rem;">Open the browser console for full details.</p>
      <pre style="background: #f3ede4; padding: 1rem; border-radius: 0.5rem; white-space: pre-wrap; font-size: 0.85rem;">${message}</pre>
    </div>
  `
}

try {
  createRoot(rootEl).render(
    <StrictMode>
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>
    </StrictMode>,
  )
} catch (error) {
  const message = error instanceof Error ? error.message : String(error)
  showBootError(rootEl, message)
  console.error(error)
}