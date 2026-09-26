// Cliente fetch fino para a API. Sempre credentials: 'include' e URL relativa
// (/api/v1/...) — o cookie access_token é HttpOnly + SameSite=Strict
// (ARCHITECTURE.md §7), então o frontend só funciona em mesma origem que a
// API (proxy do nginx em produção, proxy do Vite em dev).
const BASE_URL = "/api/v1";

export class ApiError extends Error {
  status: number;

  constructor(status: number) {
    super(`request failed with status ${status}`);
    this.status = status;
  }
}

let refreshing: Promise<boolean> | null = null;

async function refreshOnce(): Promise<boolean> {
  if (!refreshing) {
    refreshing = fetch(`${BASE_URL}/auth/refresh`, {
      method: "POST",
      credentials: "include",
    })
      .then((res) => res.ok)
      .finally(() => {
        refreshing = null;
      });
  }
  return refreshing;
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  query?: Record<string, string | boolean | undefined>;
}

function buildURL(path: string, query?: RequestOptions["query"]): string {
  const url = new URL(`${BASE_URL}${path}`, window.location.origin);
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined) url.searchParams.set(key, String(value));
    }
  }
  return url.pathname + url.search;
}

// request faz a chamada autenticada padrão. Em 401, tenta /auth/refresh uma
// única vez e refaz a chamada original; se o refresh também falhar,
// redireciona para /login — não há retry além dessa única tentativa.
async function request<T>(path: string, options: RequestOptions = {}, isRetry = false): Promise<T> {
  const res = await fetch(buildURL(path, options.query), {
    method: options.method ?? "GET",
    credentials: "include",
    headers: options.body ? { "Content-Type": "application/json" } : undefined,
    body: options.body ? JSON.stringify(options.body) : undefined,
  });

  if (res.status === 401 && !isRetry) {
    const refreshed = await refreshOnce();
    if (refreshed) {
      return request<T>(path, options, true);
    }
    window.location.assign("/login");
    throw new ApiError(401);
  }

  if (!res.ok) {
    throw new ApiError(res.status);
  }

  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

export const api = {
  get: <T>(path: string, query?: RequestOptions["query"]) => request<T>(path, { query }),
  post: <T>(path: string, body?: unknown) => request<T>(path, { method: "POST", body }),
  put: <T>(path: string, body?: unknown) => request<T>(path, { method: "PUT", body }),
};
