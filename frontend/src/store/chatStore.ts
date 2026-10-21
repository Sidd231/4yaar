import { create } from 'zustand';
import { Room, ChatMessage } from '../types/chat';
import { chatApi } from '../api/chat';

interface ChatState {
  rooms: Room[];
  activeRoomId: string | null;
  messages: Record<string, ChatMessage[]>;
  isLoadingRooms: boolean;
  isLoadingMessages: boolean;
  hasMoreMessages: Record<string, boolean>;
  wsConnected: boolean;
  error: string | null;

  setWsConnected: (connected: boolean) => void;
  setActiveRoomId: (roomId: string | null) => void;
  fetchRooms: () => Promise<void>;
  createRoom: (name: string, type?: 'group' | 'direct') => Promise<Room>;
  fetchMessages: (roomId: string, loadOlder?: boolean) => Promise<void>;
  addIncomingMessage: (msg: ChatMessage) => void;
  clearError: () => void;
}

export const useChatStore = create<ChatState>((set, get) => ({
  rooms: [],
  activeRoomId: null,
  messages: {},
  isLoadingRooms: false,
  isLoadingMessages: false,
  hasMoreMessages: {},
  wsConnected: false,
  error: null,

  setWsConnected: (connected: boolean) => set({ wsConnected: connected }),

  setActiveRoomId: (roomId: string | null) => {
    set({ activeRoomId: roomId });
  },

  clearError: () => set({ error: null }),

  fetchRooms: async () => {
    set({ isLoadingRooms: true, error: null });
    try {
      const rooms = await chatApi.listRooms();
      set({ rooms, isLoadingRooms: false });
    } catch (err: any) {
      set({
        isLoadingRooms: false,
        error: err.response?.data?.error || 'Failed to load rooms',
      });
    }
  },

  createRoom: async (name: string, type: 'group' | 'direct' = 'group') => {
    try {
      const newRoom = await chatApi.createRoom(name, type);
      set((state) => ({
        rooms: [newRoom, ...state.rooms.filter((r) => r.id !== newRoom.id)],
      }));
      return newRoom;
    } catch (err: any) {
      const msg = err.response?.data?.error || 'Failed to create room';
      set({ error: msg });
      throw err;
    }
  },

  fetchMessages: async (roomId: string, loadOlder = false) => {
    const state = get();
    const currentMessages = state.messages[roomId] || [];

    let cursor: string | undefined = undefined;
    if (loadOlder && currentMessages.length > 0) {
      cursor = currentMessages[0].created_at;
    }

    set({ isLoadingMessages: true });
    try {
      const fetched = await chatApi.getMessages(roomId, cursor, 50);

      // Backend returns records ordered by created_at ASC or DESC.
      // Ensure sorted ascending by created_at for chat timeline.
      const sorted = [...fetched].sort(
        (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
      );

      set((s) => {
        const existing = s.messages[roomId] || [];
        // Deduplicate messages by ID
        const existingMap = new Map<string, ChatMessage>();

        if (loadOlder) {
          // Prepend older messages
          [...sorted, ...existing].forEach((m) => existingMap.set(m.id, m));
        } else {
          // Initial or refresh load
          [...existing, ...sorted].forEach((m) => existingMap.set(m.id, m));
        }

        const merged = Array.from(existingMap.values()).sort(
          (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
        );

        return {
          messages: {
            ...s.messages,
            [roomId]: merged,
          },
          hasMoreMessages: {
            ...s.hasMoreMessages,
            [roomId]: fetched.length === 50,
          },
          isLoadingMessages: false,
        };
      });
    } catch (err: any) {
      set({
        isLoadingMessages: false,
        error: err.response?.data?.error || 'Failed to load message history',
      });
    }
  },

  addIncomingMessage: (msg: ChatMessage) => {
    set((s) => {
      const roomMsgs = s.messages[msg.room_id] || [];
      // Deduplicate by ID
      if (roomMsgs.some((m) => m.id === msg.id)) {
        return s;
      }
      return {
        messages: {
          ...s.messages,
          [msg.room_id]: [...roomMsgs, msg],
        },
      };
    });
  },
}));
