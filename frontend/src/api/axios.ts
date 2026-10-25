
import axios, {
  AxiosError,
  InternalAxiosRequestConfig,
} from 'axios';

import { env } from '../lib/env';

export const apiClient = axios.create({
  baseURL: env.apiBaseUrl,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 15000,
  // Required so the browser sends the httpOnly refresh_token cookie
  // and the backend can set/clear it via Set-Cookie.
  withCredentials: true,
});

// ---------------------------------------------------------------------------
// In-memory access token store
// The access token lives ONLY here (a JS module-scope variable).
// It is never written to localStorage or sessionStorage.
// ---------------------------------------------------------------------------
let inMemoryAccessToken: string | null = null;

export const setAccessToken = (token: string | null): void => {
  inMemoryAccessToken =
    typeof token === 'string' &&
    token.trim() !== '' &&
    token !== 'undefined' &&
    token !== 'null'
      ? token
      : null;
};

export const getAccessToken = (): string | null => inMemoryAccessToken;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
const isAuthEndpoint = (url = ''): boolean =>
  /\/auth\/(login|register|refresh)(?:[/?]|$)/.test(url);

interface RefreshResponse {
  access_token?: string;
  refresh_token?: string;
}

interface RetryableRequest extends InternalAxiosRequestConfig {
  _retry?: boolean;
}

// ---------------------------------------------------------------------------
// Token refresh with queue — prevents multiple concurrent refresh calls.
// ---------------------------------------------------------------------------
let refreshPromise: Promise<string | null> | null = null;

export const refreshAccessToken = async (): Promise<string | null> => {
  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = (async (): Promise<string | null> => {
    try {
      // POST with empty body — the httpOnly cookie carries the refresh token.
      // We use a plain axios instance so this call is NOT intercepted by our
      // response interceptor (avoids infinite recursion on 401).
      const response = await axios.post<{ data?: RefreshResponse } & RefreshResponse>(
        `${env.apiBaseUrl}/auth/refresh`,
        {},
        {
          timeout: 15000,
          withCredentials: true, // send the refresh_token cookie
        }
      );

      // Backend wraps responses as { data: { access_token, ... } }
      const payload = response.data?.data ?? response.data;
      const accessToken = payload?.access_token;

      if (
        typeof accessToken !== 'string' ||
        !accessToken.trim() ||
        accessToken === 'undefined' ||
        accessToken === 'null'
      ) {
        throw new Error('Refresh response did not contain a valid access_token');
      }

      setAccessToken(accessToken);
      return accessToken;
    } catch (error) {
      setAccessToken(null);

      const status = axios.isAxiosError(error)
        ? error.response?.status
        : undefined;

      console.error(
        'Access token refresh failed:',
        status ?? 'network or response error'
      );

      // On a real auth rejection redirect to login.
      if (status === 401 || status === 403) {
        window.location.href = '/login';
      }

      return null;
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
};

// ---------------------------------------------------------------------------
// Request interceptor — attach Bearer token to every protected request.
// ---------------------------------------------------------------------------
apiClient.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  if (!isAuthEndpoint(config.url) && inMemoryAccessToken) {
    if (config.headers && typeof (config.headers as any).set === 'function') {
      (config.headers as any).set('Authorization', `Bearer ${inMemoryAccessToken}`);
    } else {
      config.headers = {
        ...(config.headers ?? {}),
        Authorization: `Bearer ${inMemoryAccessToken}`,
      } as any;
    }
  }

  return config;
});

// ---------------------------------------------------------------------------
// Response interceptor — on 401 attempt one token refresh then retry.
// ---------------------------------------------------------------------------
apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as RetryableRequest | undefined;

    if (
      error.response?.status !== 401 ||
      !originalRequest ||
      originalRequest._retry ||
      isAuthEndpoint(originalRequest.url)
    ) {
      return Promise.reject(error);
    }

    originalRequest._retry = true;

    const newAccessToken = await refreshAccessToken();

    if (!newAccessToken) {
      // refreshAccessToken already redirected to /login if the server
      // rejected the refresh token; nothing more to do here.
      return Promise.reject(error);
    }

    // Retry the original request with the new access token.
    if (typeof (originalRequest.headers as any).set === 'function') {
      (originalRequest.headers as any).set('Authorization', `Bearer ${newAccessToken}`);
    } else {
      (originalRequest.headers as any)['Authorization'] = `Bearer ${newAccessToken}`;
    }

    return apiClient(originalRequest);
  }
);
