import { lazy, Suspense, Component } from "react";
import type { ReactNode } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "@/components/ui/sonner";
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
const CardsPage = lazy(() =>
  import("@/pages/cards").then((m) => ({ default: m.CardsPage }))
);
const CardDetailPage = lazy(() =>
  import("@/pages/card-detail").then((m) => ({ default: m.CardDetailPage }))
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
          path="cards"
          element={
            <Suspense fallback={<LoadingSkeleton />}>
              <CardsPage />
            </Suspense>
          }
        />
        <Route
          path="cards/:id"
          element={
            <Suspense fallback={<LoadingSkeleton />}>
              <CardDetailPage />
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

class ErrorBoundary extends Component<
  { children: ReactNode },
  { error: Error | null }
> {
  state: { error: Error | null } = { error: null };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  render() {
    if (this.state.error) {
      return (
        <div className="flex h-screen flex-col items-center justify-center gap-4 p-4">
          <h1 className="text-xl font-bold">Something went wrong</h1>
          <p className="text-sm text-muted-foreground">
            {this.state.error.message}
          </p>
          <button
            className="rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground"
            onClick={() => {
              this.setState({ error: null });
              window.location.href = "/admin/";
            }}
          >
            Go to Dashboard
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}

export default function App() {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <BrowserRouter basename="/admin">
            <Routes>
              <Route path="/*" element={<AuthGate />} />
            </Routes>
          </BrowserRouter>
          <Toaster />
        </AuthProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  );
}
