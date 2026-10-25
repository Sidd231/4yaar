
// ---------------------------------------------------------------------------
// storage.ts
//
// The refresh token is now stored exclusively as an httpOnly cookie set by
// the backend. This module NO LONGER stores the refresh token in localStorage.
//
// The access token is kept in memory by the API client (axios.ts) and is
// never persisted to storage.
//
// This file is kept for backward compatibility with any code that calls
// storage.clearAllTokens() on logout, but all token storage methods are now
// no-ops or read from localStorage only for the migration period.
// ---------------------------------------------------------------------------

export const storage = {
  // ---- Refresh token -------------------------------------------------------
  // Now managed exclusively by the httpOnly cookie from the backend.
  // These methods are intentionally no-ops on the client side.

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  setRefreshToken(_token: string | null | undefined): void {
    // Intentionally empty — the backend sets the httpOnly cookie.
  },

  clearRefreshToken(): void {
    // Intentionally empty — the backend clears the cookie on logout/refresh-fail.
  },

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  getRefreshToken(): string | null {
    // The refresh token is in an httpOnly cookie and is NOT readable from JS.
    return null;
  },

  // ---- Access token --------------------------------------------------------
  // Kept in memory only (see axios.ts). No localStorage persistence.

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  setAccessToken(_token: string | null | undefined): void {
    // Intentionally empty — managed in memory by axios.ts.
  },

  clearAccessToken(): void {
    // Intentionally empty — managed in memory by axios.ts.
  },

  getAccessToken(): string | null {
    return null;
  },

  // ---- Convenience ---------------------------------------------------------
  clearAllTokens(): void {
    // In-memory token is cleared by calling setAccessToken(null) in axios.ts.
    // httpOnly cookie is cleared by the backend on logout.
    // Nothing to clear in localStorage now.
  },
};
