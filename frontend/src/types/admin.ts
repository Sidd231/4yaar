export interface UserListItem {
  id: string;
  name: string;
  email: string;
  role: string;
  created_at: string;
}

export interface Invite {
  id: string;
  email: string;
  token: string;
  inviter_id: string;
  expires_at: string;
  used_at?: string | null;
  created_at: string;
}
