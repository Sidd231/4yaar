
import { apiClient, setAccessToken } from './axios';
import { AuthResponse, User } from '../types/auth';

// ---------------------------------------------------------------------------
// authApi
//
// All endpoints use apiClient (withCredentials: true).
// After login/register the backend sets the httpOnly refresh_token cookie.
// The access_token from the JSON body is stored in memory via setAccessToken.
// ---------------------------------------------------------------------------

function validateAuthResponse(res: AuthResponse): AuthResponse {
  if (
    !res ||
    typeof res.access_token !== 'string' ||
    !res.access_token.trim() ||
    res.access_token === 'undefined' ||
    res.access_token === 'null'
  ) {
    throw new Error(
      'Authentication response is missing a valid access_token. Check the backend response.'
    );
  }

  return res;
}

export const authApi = {
  async login(email: string, password: string): Promise<AuthResponse> {
    try {
      const response = await apiClient.post<AuthResponse>('/auth/login', {
        email,
        password,
      });

      // Backend wraps success as { data: AuthResponse, error: null }
      const body = (response.data as any)?.data ?? response.data;
      const res = validateAuthResponse(body as AuthResponse);

      setAccessToken(res.access_token);
      return res;
    } catch (error) {
      setAccessToken(null);
      throw error;
    }
  },

  async register(
    inviteToken: string,
    name: string,
    password: string
  ): Promise<AuthResponse> {
    try {
      const response = await apiClient.post<AuthResponse>('/auth/register', {
        invite_token: inviteToken,
        name,
        password,
      });

      const body = (response.data as any)?.data ?? response.data;
      const res = validateAuthResponse(body as AuthResponse);

      setAccessToken(res.access_token);
      return res;
    } catch (error) {
      setAccessToken(null);
      throw error;
    }
  },

  async getMe(): Promise<User> {
    const response = await apiClient.get<{ data?: User } & User>('/auth/me');
    return (response.data as any)?.data ?? response.data;
  },

  async logout(): Promise<void> {
    try {
      // POST with no body — the backend reads the httpOnly cookie.
      await apiClient.post('/auth/logout', {});
    } catch {
      // Always clear local state even if the server is unreachable.
    } finally {
      setAccessToken(null);
    }
  },
};
