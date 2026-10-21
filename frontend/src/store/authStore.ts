
import { create } from 'zustand';

import { User } from '../types/auth';
import { authApi } from '../api/auth';
import {
  refreshAccessToken,
  setAccessToken,
} from '../api/axios';
import { storage } from '../lib/storage';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  isInitializing: boolean;
  error: string | null;

  initAuth: () => Promise<void>;
  login: (email: string, pass: string) => Promise<void>;
  registerWithInvite: (
    token: string,
    name: string,
    pass: string
  ) => Promise<void>;
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

  initAuth: async () => {
    set({ isInitializing: true });

    try {
      const refreshToken = storage.getRefreshToken();

      if (!refreshToken) {
        setAccessToken(null);
        storage.clearAccessToken();

        set({
          user: null,
          accessToken: null,
          isAuthenticated: false,
        });

        return;
      }

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
      storage.clearAccessToken();

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
