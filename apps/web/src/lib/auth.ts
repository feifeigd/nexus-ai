const TOKEN_KEY = 'nexus_web_token';
const USER_KEY = 'nexus_web_user';

export interface AuthUser {
  username: string;
  role: string;
}

function storage(): Storage | null {
  if (typeof window === 'undefined') return null;
  return window.localStorage;
}

export function setAuth(token: string, user: AuthUser): void {
  storage()?.setItem(TOKEN_KEY, token);
  storage()?.setItem(USER_KEY, JSON.stringify(user));
}

export function getToken(): string | null {
  return storage()?.getItem(TOKEN_KEY) ?? null;
}

export function getAuthUser(): AuthUser | null {
  const raw = storage()?.getItem(USER_KEY);
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as AuthUser;
    if (typeof parsed.username === 'string' && typeof parsed.role === 'string') {
      return parsed;
    }
    return null;
  } catch {
    return null;
  }
}

export function clearAuth(): void {
  storage()?.removeItem(TOKEN_KEY);
  storage()?.removeItem(USER_KEY);
}
