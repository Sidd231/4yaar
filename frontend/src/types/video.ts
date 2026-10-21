export type SignalMessageType =
  | 'join'
  | 'offer'
  | 'answer'
  | 'ice-candidate'
  | 'leave'
  | 'peer-joined'
  | 'peer-left'
  | 'existing-peers';

export interface SignalMessage {
  type: SignalMessageType;
  room_id?: string;
  sender_id?: string;
  target_id?: string;
  data?: any;
}

export interface VideoRoomInfo {
  id: string;
  name: string;
  created_at: string;
  peer_count: number;
}
