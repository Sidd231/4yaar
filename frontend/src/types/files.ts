export interface UploadFileResponse {
  id: string;
  name: string;
  size: number;
  mime_type: string;
  s3_key: string;
  created_at: string;
}

export interface PresignedURLResponse {
  file_id: string;
  presigned_url: string;
}
