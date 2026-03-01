const API_BASE = "/admin/api";

export class AuthError extends Error {
  constructor() {
    super("Session expired");
    this.name = "AuthError";
  }
}

export async function apiFetch<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });

  if (res.status === 401) {
    throw new AuthError();
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `HTTP ${res.status}`);
  }

  return res.json();
}

export function apiPost<T>(path: string, body?: unknown): Promise<T> {
  return apiFetch(path, {
    method: "POST",
    body: body ? JSON.stringify(body) : undefined,
  });
}

export function apiPut<T>(path: string, body?: unknown): Promise<T> {
  return apiFetch(path, {
    method: "PUT",
    body: body ? JSON.stringify(body) : undefined,
  });
}
