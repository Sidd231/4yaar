import { apiClient } from './axios';
import { VideoRoomInfo } from '../types/video';

export const videoApi = {
  async getOrCreateRoom(name?: string, existingRoomId?: string): Promise<{ room_id: string; name: string }> {
    const payload: { name?: string; room_id?: string } = {};
    if (existingRoomId) payload.room_id = existingRoomId;
    if (name) payload.name = name;

    const res = await apiClient.post<{ room_id: string; name: string }>('/video/rooms', payload);
    return res.data;
  },

  async getRoomInfo(roomId: string): Promise<VideoRoomInfo> {
    const res = await apiClient.get<VideoRoomInfo>(`/video/rooms/${roomId}`);
    return res.data;
  },
};
