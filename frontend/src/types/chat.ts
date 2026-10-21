export type RoomType = 'group' | 'direct';

export interface RoomMember {
  room_id: string;
  user_id: string;
  user?: {
    id: string;
    name: string;
    email: string;
    role: string;
  };
  joined_at: string;
}

export interface Room {
  id: string;
  name: string;
  type: RoomType;
  created_by: string;
  created_at: string;
  members?: RoomMember[];
}

export interface MessageFile {
  id: string;
  name: string;
  size: number;
  mime_type: string;
  s3_key?: string;
  created_at?: string;
}

export interface ChatMessage {
  id: string;
  room_id: string;
  sender_id: string;
  sender?: {
    id: string;
    name: string;
    email: string;
    role: string;
  };
  content: string;
  type: 'text' | 'image' | 'file';
  file_id?: string | null;
  file?: MessageFile | null;
  created_at: string;
}

export interface SendMessagePayload {
  content: string;
  type: 'text' | 'image' | 'file';
  file_id?: string | null;
}
