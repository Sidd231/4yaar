import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useChatStore } from '../store/chatStore';
import { useCallStore } from '../store/callStore';
import { videoApi } from '../api/video';
import { useChatSocket } from '../hooks/useChatSocket';
import { Sidebar } from '../components/layout/Sidebar';
import { MessageList } from '../components/chat/MessageList';
import { MessageInput } from '../components/chat/MessageInput';
import { CreateRoomDialog } from '../components/chat/CreateRoomDialog';
import { CallOverlay } from '../components/videocall/CallOverlay';
import { Phone, Hash, User as UserIcon } from 'lucide-react';

export const ChatPage: React.FC = () => {
  const { roomId } = useParams<{ roomId?: string }>();
  const navigate = useNavigate();

  const {
    rooms,
    messages,
    isLoadingMessages,
    hasMoreMessages,
    fetchRooms,
    fetchMessages,
    setActiveRoomId,
    wsConnected,
  } = useChatStore();

  const { startCall } = useCallStore();

  const [isCreateRoomOpen, setIsCreateRoomOpen] = useState(false);

  // Load rooms on initial mount
  useEffect(() => {
    fetchRooms();
  }, [fetchRooms]);

  // If on /chat and rooms exist, redirect to first room
  useEffect(() => {
    if (!roomId && rooms.length > 0) {
      navigate(`/chat/${rooms[0].id}`, { replace: true });
    }
  }, [roomId, rooms, navigate]);

  // Set active room and fetch message history
  useEffect(() => {
    if (roomId) {
      setActiveRoomId(roomId);
      fetchMessages(roomId);
    } else {
      setActiveRoomId(null);
    }
  }, [roomId, setActiveRoomId, fetchMessages]);

  // Connect chat websocket
  const { sendMessage } = useChatSocket(roomId || null);

  const currentRoom = rooms.find((r) => r.id === roomId);
  const roomMessages = roomId ? messages[roomId] || [] : [];
  const hasMore = roomId ? !!hasMoreMessages[roomId] : false;

  const handleStartVideoCall = async () => {
    if (!roomId || !currentRoom) return;
    try {
      // Ensure backend video room exists or is created
      const videoRoom = await videoApi.getOrCreateRoom(currentRoom.name, roomId);
      startCall(videoRoom.room_id, currentRoom.name);
    } catch {
      // Fallback: start call with current room ID directly
      startCall(roomId, currentRoom.name);
    }
  };

  return (
    <div className="h-screen w-screen flex bg-brand-black overflow-hidden font-sans">
      {/* Sidebar */}
      <Sidebar onOpenCreateRoom={() => setIsCreateRoomOpen(true)} />

      {/* Main Chat Area */}
      <main className="flex-1 ml-60 flex flex-col h-full bg-brand-black">
        {roomId && currentRoom ? (
          <>
            {/* Top Bar Header */}
            <header className="h-14 border-b border-brand-border bg-brand-surface/60 backdrop-blur-xs px-6 flex items-center justify-between shrink-0 select-none">
              <div className="flex items-center gap-3">
                {currentRoom.type === 'direct' ? (
                  <UserIcon className="w-4 h-4 text-brand-textMuted" />
                ) : (
                  <Hash className="w-4 h-4 text-brand-amber" />
                )}
                <div>
                  <h2 className="text-sm font-semibold text-brand-textHeading leading-none">
                    {currentRoom.name}
                  </h2>
                  <span className="text-[10px] font-mono text-brand-textMuted">
                    {wsConnected ? (
                      <span className="text-emerald-500">● live</span>
                    ) : (
                      <span className="text-zinc-500">connecting...</span>
                    )}
                  </span>
                </div>
              </div>

              <div className="flex items-center gap-2">
                <button
                  onClick={handleStartVideoCall}
                  className="px-3 py-1.5 rounded bg-brand-card hover:bg-zinc-800 text-brand-textHeading border border-brand-border text-xs font-mono font-medium flex items-center gap-1.5 transition-colors"
                >
                  <Phone className="w-3.5 h-3.5 text-brand-amber" />
                  <span>Start Call</span>
                </button>
              </div>
            </header>

            {/* Message Timeline */}
            <MessageList
              messages={roomMessages}
              isLoading={isLoadingMessages}
              hasMore={hasMore}
              onLoadOlder={() => roomId && fetchMessages(roomId, true)}
            />

            {/* Message Composer */}
            <MessageInput
              roomId={roomId}
              onSendMessage={sendMessage}
              disabled={!wsConnected}
            />
          </>
        ) : (
          <div className="flex-1 flex flex-col items-center justify-center p-6 text-center">
            <div className="w-10 h-10 rounded-full bg-brand-surface border border-brand-border flex items-center justify-center mb-3">
              <Hash className="w-5 h-5 text-brand-amber" />
            </div>
            <h3 className="text-sm font-semibold text-brand-textHeading mb-1">
              Select or create a room
            </h3>
            <p className="text-xs text-brand-textMuted max-w-sm mb-4">
              Choose a channel from the sidebar or create a new room to start collaborating.
            </p>
            <button
              onClick={() => setIsCreateRoomOpen(true)}
              className="px-3 py-1.5 rounded bg-brand-amber text-brand-black text-xs font-semibold hover:bg-amber-400 transition-colors"
            >
              + Create Room
            </button>
          </div>
        )}
      </main>

      {/* Persistent Call Overlay */}
      <CallOverlay />

      {/* Dialogs */}
      <CreateRoomDialog
        isOpen={isCreateRoomOpen}
        onClose={() => setIsCreateRoomOpen(false)}
      />
    </div>
  );
};
