import { apiClient } from './axios';
import { UploadFileResponse, PresignedURLResponse } from '../types/files';

export const filesApi = {
  async uploadFile(file: File, roomId: string, onProgress?: (pct: number) => void): Promise<UploadFileResponse> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('room_id', roomId);

    const res = await apiClient.post<UploadFileResponse>('/files/upload', formData, {
      headers: {
        // Axios will automatically set multipart/form-data with boundary when data is FormData
        'Content-Type': undefined,
      },
      onUploadProgress: (progressEvent) => {
        if (progressEvent.total && onProgress) {
          const pct = Math.round((progressEvent.loaded * 100) / progressEvent.total);
          onProgress(pct);
        }
      },
    });
    return res.data;
  },

  async getDownloadUrl(fileId: string): Promise<PresignedURLResponse> {
    const res = await apiClient.get<PresignedURLResponse>(`/files/${fileId}`);
    return res.data;
  },

  async deleteFile(fileId: string): Promise<void> {
    await apiClient.delete(`/files/${fileId}`);
  },
};
