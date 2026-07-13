import { lazy, Suspense, type ComponentType } from "react";
import { createBrowserRouter, Navigate } from "react-router-dom";

import { AuthGuard } from "@/components/auth/auth-guard";
import { GuestGuard } from "@/components/auth/guest-guard";
import { AuthLayout } from "@/components/layout/auth-layout";
import { ProtectedLayout } from "@/components/layout/protected-layout";
import { AuthTransitionLoader } from "@/components/auth/auth-transition-loader";

const HomePage = lazy(() =>
  import("@/pages/home-page").then((m) => ({ default: m.HomePage })),
);

const MarketsPage = lazy(() =>
  import("@/pages/markets-page").then((m) => ({ default: m.MarketsPage })),
);

const OpenWagersPage = lazy(() =>
  import("@/pages/open-wagers-page").then((m) => ({
    default: m.OpenWagersPage,
  })),
);
const MyWagersPage = lazy(() =>
  import("@/pages/my-wagers-page").then((m) => ({ default: m.MyWagersPage })),
);
const LoginPage = lazy(() =>
  import("@/pages/login-page").then((m) => ({ default: m.LoginPage })),
);
const AuthVerifyPage = lazy(() =>
  import("@/pages/auth-verify-page").then((m) => ({
    default: m.AuthVerifyPage,
  })),
);
const InvitesPage = lazy(() =>
  import("@/pages/invites-page").then((m) => ({ default: m.InvitesPage })),
);
const InviteDetailPage = lazy(() =>
  import("@/pages/invite-detail-page").then((m) => ({
    default: m.InviteDetailPage,
  })),
);
const WagerDetailPage = lazy(() =>
  import("@/pages/wager-detail-page").then((m) => ({
    default: m.WagerDetailPage,
  })),
);
const ProfilePage = lazy(() =>
  import("@/pages/profile-page").then((m) => ({ default: m.ProfilePage })),
);
const HistoryPage = lazy(() =>
  import("@/pages/history-page").then((m) => ({ default: m.HistoryPage })),
);

const TermsPage = lazy(() =>
  import("@/pages/terms-page").then((m) => ({ default: m.TermsPage })),
);

const LeaderboardPage = lazy(() =>
  import("@/pages/leaderboard-page").then((m) => ({
    default: m.LeaderboardPage,
  })),
);

function PageLoader() {
  return <AuthTransitionLoader />;
}

function withSuspense(Page: ComponentType) {
  return (
    <Suspense fallback={<PageLoader />}>
      <Page />
    </Suspense>
  );
}

export const router = createBrowserRouter([
  {
    path: "/",
    element: withSuspense(HomePage),
  },
  {
    path: "/terms",
    element: withSuspense(TermsPage),
  },
  {
    element: <GuestGuard />,
    children: [
      {
        element: <AuthLayout showHeader={false} showFooter={false} />,
        children: [{ path: "/login", element: withSuspense(LoginPage) }],
      },
    ],
  },
  {
    path: "/auth/verify",
    element: <AuthLayout showFooter={false} showHeader={false} />,
    children: [{ index: true, element: withSuspense(AuthVerifyPage) }],
  },

  {
    element: <AuthGuard />,
    children: [
      {
        element: <ProtectedLayout />,
        children: [
          { index: true, element: <Navigate to="/markets" replace /> },
          { path: "markets", element: withSuspense(MarketsPage) },

          { path: "open", element: withSuspense(OpenWagersPage) },
          { path: "my-wagers", element: withSuspense(MyWagersPage) },
          {
            path: "my-wagers/:wagerPubkey",
            element: withSuspense(WagerDetailPage),
          },
          { path: "history", element: withSuspense(HistoryPage) },
          { path: "invites", element: withSuspense(InvitesPage) },
          { path: "invites/:id", element: withSuspense(InviteDetailPage) },
          { path: "leaderboard", element: withSuspense(LeaderboardPage) },
          { path: "profile", element: withSuspense(ProfilePage) },
          { path: "*", element: <Navigate to="/markets" replace /> },
        ],
      },
    ],
  },
]);
