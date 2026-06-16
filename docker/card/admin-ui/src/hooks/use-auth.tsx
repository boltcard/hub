import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from "react";
import { apiFetch, apiPost } from "@/lib/api";

export class TotpRequiredError extends Error {
  constructor() {
    super("2fa required");
    this.name = "TotpRequiredError";
  }
}

interface AuthState {
  loading: boolean;
  authenticated: boolean;
  registered: boolean;
}

interface AuthContextType extends AuthState {
  login: (password: string, code?: string) => Promise<void>;
  register: (password: string) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({
    loading: true,
    authenticated: false,
    registered: false,
  });

  const refresh = useCallback(async () => {
    try {
      const data = await apiFetch<{ authenticated: boolean; registered: boolean }>(
        "/auth/check"
      );
      setState({ loading: false, ...data });
    } catch {
      setState({ loading: false, authenticated: false, registered: false });
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const login = async (password: string, code?: string) => {
    const res = await fetch("/admin/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ password, code }),
    });
    if (res.ok) {
      await refresh();
      return;
    }
    const body = await res
      .json()
      .catch(() => ({}) as { error?: string; totpRequired?: boolean });
    if (body.totpRequired) {
      throw new TotpRequiredError();
    }
    throw new Error(body.error || `HTTP ${res.status}`);
  };

  const register = async (password: string) => {
    await apiPost("/auth/register", { password });
    setState((s) => ({ ...s, registered: true }));
  };

  const logout = async () => {
    await apiPost("/auth/logout");
    setState({ loading: false, authenticated: false, registered: true });
  };

  return (
    <AuthContext.Provider value={{ ...state, login, register, logout, refresh }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
