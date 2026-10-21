export interface User {
  id: string;
  name: string;
  email: string;
  role: 'admin' | 'member' | string;
  created_at: string;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface InviteVerifyResponse {
  valid: boolean;
  email: string;
  expires_at: string;
}
