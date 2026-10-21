
import { apiClient, setAccessToken } from './axios';
import { AuthResponse, User } from '../types/auth';
import { storage } from '../lib/storage';

function validateAuthResponse(res: AuthResponse): AuthResponse {
  if (
    !res ||
    typeof res.access_token !== 'string' ||
    !res.access_token.trim() ||
    res.access_token === 'undefined' ||
    res.access_token === 'null' ||
    typeof res.refresh_token !== 'string' ||
    !res.refresh_token.trim() ||
    res.refresh_token === 'undefined' ||
    res.refresh_token === 'null'
  ) {
    throw new Error(
      'Authentication response is missing valid tokens. Check the backend response.'
    );
  }

  return res;
}

export const authApi = {
  async login(
    email: string,
    password: string
  ): Promise<AuthResponse> {
    try {
      const response = await apiClient.post<AuthResponse>(
        '/auth/login',
        { email, password }
      );

      const res = validateAuthResponse(response.data);

      setAccessToken(res.access_token);
      storage.setAccessToken(res.access_token);
      storage.setRefreshToken(res.refresh_token);

      return res;
    } catch (error) {
      // Do not leave a previous session active after a failed login.
      setAccessToken(null);
      storage.clearAccessToken();
      storage.clearRefreshToken();

      throw error;
    }
  },

  async register(
    inviteToken: string,
    name: string,
    password: string
  ): Promise<AuthResponse> {
    try {
      const response = await apiClient.post<AuthResponse>(
        '/auth/register',
        {
          invite_token: inviteToken,
          name,
          password,
        }
      );

      const res = validateAuthResponse(response.data);

      setAccessToken(res.access_token);
      storage.setAccessToken(res.access_token);
      storage.setRefreshToken(res.refresh_token);

      return res;
    } catch (error) {
      setAccessToken(null);
      storage.clearAccessToken();
      storage.clearRefreshToken();

      throw error;
    }
  },

  async getMe(): Promise<User> {
    const response = await apiClient.get<User>('/auth/me');
    return response.data;
  },

  async logout(): Promise<void> {
    const refreshToken = storage.getRefreshToken();

    try {
      if (refreshToken) {
        await apiClient.post('/auth/logout', {
          refresh_token: refreshToken,
        });
      }
    } catch {
      // Logout locally even if the backend is unreachable.
    } finally {
      setAccessToken(null);
      storage.clearAllTokens();
    }
  },
};
