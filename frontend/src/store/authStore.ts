
import { create } from 'zustand';

import { User } from '../types/auth';
import { authApi } from '../api/auth';
import { refreshAccessToken, setAccessToken } from '../api/axios';

// ---------------------------------------------------------------------------
// Auth state — Zustand store
//
// Token storage strategy:
//  - access token  → in-memory variable inside axios.ts (no localStorage)
//  - refresh token → httpOnly cookie set by the backend (not readable from JS)
//
// On app boot (initAuth) we attempt a silent token refresh. If the browser
// holds a valid httpOnly refresh_token cookie the backend issues a new access
// token; otherwise the user must log in again.
// ---------------------------------------------------------------------------

interface AuthState {
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  isInitializing: boolean;
  error: string | null;

  initAuth: () => Promise<void>;
  login: (email: string, pass: string) => Promise<void>;
  registerWithInvite: (token: string, name: string, pass: string) => Promise<void>;
  logout: () => Promise<void>;
  clearError: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  accessToken: null,
  isAuthenticated: false,
  isInitializing: true,
  error: null,

  clearError: () => set({ error: null }),

  // -------------------------------------------------------------------------
  // initAuth — called once on app mount (App.tsx useEffect).
  // Silently tries to get a new access token via the refresh_token cookie.
  // -------------------------------------------------------------------------
  initAuth: async () => {
    set({ isInitializing: true });

    try {
      // Attempt silent refresh. The httpOnly cookie is sent automatically.
      // If no cookie exists (or it's expired) the backend returns 401 and
      // refreshAccessToken() returns null.
      const newAccessToken = await refreshAccessToken();

      if (!newAccessToken) {
        set({
          user: null,
          accessToken: null,
          isAuthenticated: false,
        });
        return;
      }

      setAccessToken(newAccessToken);

      const user = await authApi.getMe();

      set({
        user,
        accessToken: newAccessToken,
        isAuthenticated: true,
        error: null,
      });
    } catch {
      setAccessToken(null);

      set({
        user: null,
        accessToken: null,
        isAuthenticated: false,
      });
    } finally {
      set({ isInitializing: false });
    }
  },

  login: async (email, pass) => {
    set({ error: null });

    try {
      const res = await authApi.login(email, pass);

      set({
        user: res.user,
        accessToken: res.access_token,
        isAuthenticated: true,
        isInitializing: false,
        error: null,
      });
    } catch (error: unknown) {
      const msg =
        (error as any)?.response?.data?.error ||
        (error instanceof Error ? error.message : null) ||
        'Login failed';

      set({
        user: null,
        accessToken: null,
        isAuthenticated: false,
        error: msg,
      });

      throw error;
    }
  },

  registerWithInvite: async (token, name, pass) => {
    set({ error: null });

    try {
      const res = await authApi.register(token, name, pass);

      set({
        user: res.user,
        accessToken: res.access_token,
        isAuthenticated: true,
        isInitializing: false,
        error: null,
      });
    } catch (error: unknown) {
      const msg =
        (error as any)?.response?.data?.error ||
        (error instanceof Error ? error.message : null) ||
        'Registration failed';

      set({
        user: null,
        accessToken: null,
        isAuthenticated: false,
        error: msg,
      });

      throw error;
    }
  },

  logout: async () => {
    try {
      await authApi.logout();
    } finally {
      set({
        user: null,
        accessToken: null,
        isAuthenticated: false,
        isInitializing: false,
        error: null,
      });
    }
  },
}));
