import { apiClient } from './axios';
import { UserListItem } from '../types/admin';

export const usersApi = {
  async listUsers(search = ''): Promise<UserListItem[]> {
    const params = new URLSearchParams();
    if (search.trim()) params.set('search', search.trim());
    const query = params.toString() ? `?${params.toString()}` : '';
    const res = await apiClient.get<UserListItem[]>(`/users${query}`);
    return res.data;
  },
};
