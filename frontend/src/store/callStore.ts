import { create } from 'zustand';

export type CallStatus = 'idle' | 'connecting' | 'connected' | 'disconnected' | 'error';

interface CallState {
  activeCallRoomId: string | null;
  activeCallRoomName: string | null;
  status: CallStatus;
  isMuted: boolean;
  isCameraOff: boolean;
  isScreenSharing: boolean;
  isMinimized: boolean;
  overlayPosition: { x: number; y: number };
  peers: string[]; // List of peer user IDs
  error: string | null;

  startCall: (roomId: string, roomName?: string) => void;
  setStatus: (status: CallStatus) => void;
  setMuted: (isMuted: boolean) => void;
  setCameraOff: (isCameraOff: boolean) => void;
  setScreenSharing: (isScreenSharing: boolean) => void;
  setMinimized: (isMinimized: boolean) => void;
  setOverlayPosition: (pos: { x: number; y: number }) => void;
  setPeers: (peers: string[]) => void;
  addPeer: (peerId: string) => void;
  removePeer: (peerId: string) => void;
  setError: (err: string | null) => void;
  endCall: () => void;
}

export const useCallStore = create<CallState>((set) => ({
  activeCallRoomId: null,
  activeCallRoomName: null,
  status: 'idle',
  isMuted: false,
  isCameraOff: false,
  isScreenSharing: false,
  isMinimized: false,
  overlayPosition: { x: 24, y: 80 },
  peers: [],
  error: null,

  startCall: (roomId: string, roomName?: string) => {
    set({
      activeCallRoomId: roomId,
      activeCallRoomName: roomName || 'Video Call',
      status: 'connecting',
      isMinimized: false,
      error: null,
      peers: [],
    });
  },

  setStatus: (status) => set({ status }),
  setMuted: (isMuted) => set({ isMuted }),
  setCameraOff: (isCameraOff) => set({ isCameraOff }),
  setScreenSharing: (isScreenSharing) => set({ isScreenSharing }),
  setMinimized: (isMinimized) => set({ isMinimized }),
  setOverlayPosition: (pos) => set({ overlayPosition: pos }),
  setPeers: (peers) => set({ peers }),
  addPeer: (peerId) =>
    set((state) => ({
      peers: state.peers.includes(peerId) ? state.peers : [...state.peers, peerId],
    })),
  removePeer: (peerId) =>
    set((state) => ({
      peers: state.peers.filter((id) => id !== peerId),
    })),
  setError: (error) => set({ error }),
  endCall: () => {
    set({
      activeCallRoomId: null,
      activeCallRoomName: null,
      status: 'idle',
      isScreenSharing: false,
      peers: [],
      error: null,
    });
  },
}));
