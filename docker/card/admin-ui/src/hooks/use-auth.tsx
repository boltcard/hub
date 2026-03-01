import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from "react";
import { apiFetch, apiPost } from "@/lib/api";

interface AuthState {
  loading: boolean;
  authenticated: boolean;
  registered: boolean;
}

interface AuthContextType extends AuthState {
  login: (password: string) => Promise<void>;
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

  const login = async (password: string) => {
    await apiPost("/auth/login", { password });
    await refresh();
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
