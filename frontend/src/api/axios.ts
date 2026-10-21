
import axios, {
  AxiosError,
  InternalAxiosRequestConfig,
} from 'axios';

import { env } from '../lib/env';
import { storage } from '../lib/storage';

export const apiClient = axios.create({
  baseURL: env.apiBaseUrl,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 15000,
});

// Keep the access token in memory.
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

export const getAccessToken = (): string | null => {
  return inMemoryAccessToken;
};

const isAuthEndpoint = (url = ''): boolean =>
  /\/auth\/(login|register|refresh)(?:[/?]|$)/.test(url);

interface RefreshResponse {
  access_token?: string;
  refresh_token?: string;
}

interface RetryableRequest extends InternalAxiosRequestConfig {
  _retry?: boolean;
}

let refreshPromise: Promise<string | null> | null = null;

export const refreshAccessToken = async (): Promise<string | null> => {
  // Reuse an existing refresh operation when requests fail together.
  if (refreshPromise) {
    return refreshPromise;
  }

  const refreshToken = storage.getRefreshToken();

  if (!refreshToken) {
    setAccessToken(null);
    storage.clearAccessToken();
    storage.clearRefreshToken();
    return null;
  }

  refreshPromise = (async (): Promise<string | null> => {
    try {
      // Use plain axios to avoid response-interceptor recursion.
      const response = await axios.post<{
        data?: RefreshResponse;
        access_token?: string;
        refresh_token?: string;
      }>(
        `${env.apiBaseUrl}/auth/refresh`,
        { refresh_token: refreshToken },
        { timeout: 15000 }
      );

      // Support either a direct token response or a { data: ... }
      // wrapper, depending on the backend's response format.
      const payload =
        response.data?.data ?? response.data;

      const accessToken = payload?.access_token;
      const newRefreshToken = payload?.refresh_token;

      if (
        typeof accessToken !== 'string' ||
        !accessToken.trim() ||
        accessToken === 'undefined' ||
        accessToken === 'null'
      ) {
        throw new Error(
          'Refresh response did not contain a valid access_token'
        );
      }

      setAccessToken(accessToken);

      // Some backends rotate refresh tokens; others do not.
      if (newRefreshToken !== undefined) {
        if (
          typeof newRefreshToken !== 'string' ||
          !newRefreshToken.trim() ||
          newRefreshToken === 'undefined' ||
          newRefreshToken === 'null'
        ) {
          throw new Error('Invalid refresh_token in refresh response');
        }

        storage.setRefreshToken(newRefreshToken);
      }

      return accessToken;
    } catch (error) {
      setAccessToken(null);
      storage.clearAccessToken();

      const status = axios.isAxiosError(error)
        ? error.response?.status
        : undefined;

      // Only discard the refresh token when the server rejects it.
      // Keep it if a network error or server outage prevents refresh.
      if (status === 401 || status === 403) {
        storage.clearRefreshToken();
      }

      console.error(
        'Access token refresh failed:',
        status ?? 'network or response error'
      );

      return null;
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
};

// Attach the access token to protected requests.
apiClient.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  // Load token from storage if it's not in memory yet.
  if (!inMemoryAccessToken) {
    const stored = storage.getAccessToken();
    if (stored) {
      inMemoryAccessToken = stored;
    }
  }

  if (!isAuthEndpoint(config.url) && inMemoryAccessToken) {
    // Header may be an AxiosHeaders instance (which has .set) or a plain object.
    if (config.headers && typeof (config.headers as any).set === 'function') {
      (config.headers as any).set('Authorization', `Bearer ${inMemoryAccessToken}`);
    } else {
      config.headers = {
        ...(config.headers ?? {}),
        Authorization: `Bearer ${inMemoryAccessToken}`,
      } as any;
    }
  }

  // Avoid logging the token itself.
  return config;
});

// Refresh and retry a protected request after a 401.
apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest =
      error.config as RetryableRequest | undefined;

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
      return Promise.reject(error);
    }

    originalRequest.headers.set(
      'Authorization',
      `Bearer ${newAccessToken}`
    );

    return apiClient(originalRequest);
  }
);
