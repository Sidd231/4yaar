
const REFRESH_TOKEN_KEY = '4yaar_refresh_token';
const ACCESS_TOKEN_KEY = '4yaar_access_token';

function isValidToken(
  token: unknown
): token is string {
  return (
    typeof token === 'string' &&
    token.trim().length > 0 &&
    token !== 'undefined' &&
    token !== 'null'
  );
}

export const storage = {
  getRefreshToken(): string | null {
    try {
      const token = localStorage.getItem(REFRESH_TOKEN_KEY);
      return isValidToken(token) ? token : null;
    } catch {
      return null;
    }
  },

  setRefreshToken(token: string | null | undefined): void {
    try {
      if (!isValidToken(token)) {
        localStorage.removeItem(REFRESH_TOKEN_KEY);
        return;
      }

      localStorage.setItem(REFRESH_TOKEN_KEY, token);
    } catch (error) {
      console.error('Failed to store refresh token', error);
    }
  },

  clearRefreshToken(): void {
    try {
      localStorage.removeItem(REFRESH_TOKEN_KEY);
    } catch {
      // Storage may be unavailable.
    }
  },

  // Compatibility helpers. Access tokens are kept in memory
  // by the API client in this implementation.
  getAccessToken(): string | null {
    try {
      const token = localStorage.getItem(ACCESS_TOKEN_KEY);
      return isValidToken(token) ? token : null;
    } catch {
      return null;
    }
  },

  setAccessToken(token: string | null | undefined): void {
    try {
      if (!isValidToken(token)) {
        localStorage.removeItem(ACCESS_TOKEN_KEY);
        return;
      }

      localStorage.setItem(ACCESS_TOKEN_KEY, token);
    } catch (error) {
      console.error('Failed to store access token', error);
    }
  },

  clearAccessToken(): void {
    try {
      localStorage.removeItem(ACCESS_TOKEN_KEY);
    } catch {
      // Storage may be unavailable.
    }
  },

  clearAllTokens(): void {
    this.clearAccessToken();
    this.clearRefreshToken();
  },
};
