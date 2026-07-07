import { Outlet } from 'react-router-dom'

import { OnboardingUsernameModal } from '@/components/auth/onboarding-username-modal'
import { SessionExpiredBanner } from '@/components/auth/session-expired-banner'
import { AppShell } from '@/components/layout/app-shell'

export function ProtectedLayout() {
  return (
    <AppShell>
      <SessionExpiredBanner />
      <OnboardingUsernameModal />
      <Outlet />
    </AppShell>
  )
}