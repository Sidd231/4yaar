import { apiClient } from './axios';
import { Room, ChatMessage } from '../types/chat';

export const chatApi = {
  async listRooms(): Promise<Room[]> {
    const res = await apiClient.get<Room[]>('/chat/rooms');
    return res.data;
  },

  async createRoom(name: string, type: 'group' | 'direct' = 'group'): Promise<Room> {
    const res = await apiClient.post<Room>('/chat/rooms', { name, type });
    return res.data;
  },

  async addMember(roomId: string, userId: string): Promise<void> {
    await apiClient.post(`/chat/rooms/${roomId}/members`, { user_id: userId });
  },

  async getMessages(roomId: string, cursor?: string, limit = 50): Promise<ChatMessage[]> {
    const params = new URLSearchParams();
    if (cursor) params.set('cursor', cursor);
    if (limit) params.set('limit', limit.toString());

    const queryString = params.toString() ? `?${params.toString()}` : '';
    const res = await apiClient.get<ChatMessage[]>(`/chat/rooms/${roomId}/messages${queryString}`);
    return res.data;
  },
};
