import { lazy, Suspense } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider, useAuth } from "@/hooks/use-auth";
import { AppShell } from "@/components/app-shell";
import { LoginPage } from "@/pages/login";
import { RegisterPage } from "@/pages/register";
import { DashboardPage } from "@/pages/dashboard";

const PhoenixPage = lazy(() =>
  import("@/pages/phoenix").then((m) => ({ default: m.PhoenixPage }))
);
const SettingsPage = lazy(() =>
  import("@/pages/settings").then((m) => ({ default: m.SettingsPage }))
);
const AboutPage = lazy(() =>
  import("@/pages/about").then((m) => ({ default: m.AboutPage }))
);
const DatabasePage = lazy(() =>
  import("@/pages/database").then((m) => ({ default: m.DatabasePage }))
);

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 10_000,
    },
  },
});

function LoadingSkeleton() {
  return <div className="h-64 animate-pulse rounded-lg bg-muted" />;
}

function AuthGate() {
  const { loading, authenticated, registered } = useAuth();

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="h-6 w-6 animate-pulse rounded-full bg-primary" />
      </div>
    );
  }

  if (!registered) return <RegisterPage />;
  if (!authenticated) return <LoginPage />;

  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<DashboardPage />} />
        <Route
          path="phoenix"
          element={
            <Suspense fallback={<LoadingSkeleton />}>
              <PhoenixPage />
            </Suspense>
          }
        />
        <Route
          path="settings"
          element={
            <Suspense fallback={<LoadingSkeleton />}>
              <SettingsPage />
            </Suspense>
          }
        />
        <Route
          path="about"
          element={
            <Suspense fallback={<LoadingSkeleton />}>
              <AboutPage />
            </Suspense>
          }
        />
        <Route
          path="database"
          element={
            <Suspense fallback={<LoadingSkeleton />}>
              <DatabasePage />
            </Suspense>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter basename="/admin">
          <Routes>
            <Route path="/*" element={<AuthGate />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </QueryClientProvider>
  );
}
