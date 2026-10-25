import { apiClient } from './axios';
import { InviteVerifyResponse } from '../types/auth';
import { Invite } from '../types/admin';

export const invitesApi = {
  async verify(token: string): Promise<InviteVerifyResponse> {
    const res = await apiClient.get<any>(`/invites/verify?token=${encodeURIComponent(token)}`);
    return res.data?.data !== undefined ? res.data.data : res.data;
  },

  async create(email: string): Promise<Invite> {
    const res = await apiClient.post<any>('/invites', { email });
    return res.data?.data !== undefined ? res.data.data : res.data;
  },
};
