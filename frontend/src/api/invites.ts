import { apiClient } from './axios';
import { InviteVerifyResponse } from '../types/auth';
import { Invite } from '../types/admin';

export const invitesApi = {
  async verify(token: string): Promise<InviteVerifyResponse> {
    const res = await apiClient.get<InviteVerifyResponse>(`/invite/verify?token=${encodeURIComponent(token)}`);
    return res.data;
  },

  async create(email: string): Promise<Invite> {
    const res = await apiClient.post<Invite>('/invite/', { email });
    return res.data;
  },
};
