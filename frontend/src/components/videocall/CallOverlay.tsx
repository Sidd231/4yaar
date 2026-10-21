import React, { useEffect, useRef } from 'react';
import { Mic, MicOff, Video, VideoOff, Monitor, PhoneOff, Minimize2, Maximize2 } from 'lucide-react';
import { useCallStore } from '../../store/callStore';
import { useVideoCall } from '../../hooks/useVideoCall';

export const CallOverlay: React.FC = () => {
  const {
    activeCallRoomId,
    activeCallRoomName,
    status,
    isMuted,
    isCameraOff,
    isScreenSharing,
    isMinimized,
    peers,
    setMuted,
    setCameraOff,
    setMinimized,
  } = useCallStore();

  const { localStream, getRemoteStream, toggleScreenShare, endCall } = useVideoCall();

  const localVideoRef = useRef<HTMLVideoElement>(null);
  const remoteVideoRefs = useRef<Map<string, HTMLVideoElement>>(new Map());

  // Attach local video stream
  useEffect(() => {
    if (localVideoRef.current && localStream) {
      localVideoRef.current.srcObject = localStream;
    }
  }, [localStream, isCameraOff, isScreenSharing]);

  if (!activeCallRoomId) return null;

  if (isMinimized) {
    return (
      <div className="fixed bottom-4 right-4 z-50 bg-brand-surface border border-brand-border rounded-lg shadow-2xl p-3 flex items-center gap-3 select-none">
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-brand-amber animate-pulse" />
          <span className="text-xs font-mono font-medium text-brand-textHeading truncate max-w-[120px]">
            {activeCallRoomName}
          </span>
          <span className="text-[10px] font-mono text-zinc-500">
            ({peers.length + 1} connected)
          </span>
        </div>

        <div className="flex items-center gap-1.5 border-l border-brand-border pl-2">
          <button
            onClick={() => setMuted(!isMuted)}
            className={`p-1.5 rounded transition-colors ${
              isMuted ? 'text-red-400 bg-red-500/10' : 'text-brand-textMuted hover:text-brand-textHeading'
            }`}
            title={isMuted ? 'Unmute' : 'Mute'}
          >
            {isMuted ? <MicOff className="w-3.5 h-3.5" /> : <Mic className="w-3.5 h-3.5" />}
          </button>

          <button
            onClick={() => setMinimized(false)}
            className="p-1.5 text-brand-textMuted hover:text-brand-textHeading rounded transition-colors"
            title="Expand Call"
          >
            <Maximize2 className="w-3.5 h-3.5" />
          </button>

          <button
            onClick={endCall}
            className="p-1.5 text-red-400 hover:bg-red-500/20 rounded transition-colors"
            title="Leave Call"
          >
            <PhoneOff className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed bottom-6 right-6 z-50 w-96 max-w-[calc(100vw-3rem)] bg-brand-surface border border-brand-border rounded-lg shadow-2xl overflow-hidden flex flex-col select-none">
      {/* Call Header */}
      <div className="px-3 py-2 bg-brand-card/80 border-b border-brand-border flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-brand-amber animate-pulse" />
          <span className="text-xs font-medium text-brand-textHeading font-display truncate">
            {activeCallRoomName}
          </span>
          <span className="text-[10px] font-mono text-zinc-500">
            {status === 'connected' ? `${peers.length + 1} in call` : status}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={() => setMinimized(true)}
            className="p-1 text-brand-textMuted hover:text-brand-textHeading rounded transition-colors"
            title="Minimize"
          >
            <Minimize2 className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {/* Video Grid */}
      <div className="p-2 grid grid-cols-2 gap-2 bg-brand-black min-h-[160px] max-h-[300px] overflow-y-auto">
        {/* Local participant tile */}
        <div className="relative aspect-video bg-zinc-900 rounded border border-brand-border overflow-hidden flex items-center justify-center">
          <video
            ref={localVideoRef}
            autoPlay
            playsInline
            muted
            className={`w-full h-full object-cover ${isCameraOff ? 'hidden' : ''}`}
          />
          {isCameraOff && (
            <div className="flex flex-col items-center gap-1 text-zinc-500">
              <VideoOff className="w-5 h-5" />
              <span className="text-[10px] font-mono">Camera Off</span>
            </div>
          )}
          <div className="absolute bottom-1 left-1 px-1.5 py-0.5 rounded bg-black/70 text-[9px] font-mono text-zinc-300">
            You {isMuted && '(Muted)'}
          </div>
        </div>

        {/* Remote participant tiles */}
        {peers.map((peerId) => {
          const remoteStream = getRemoteStream(peerId);
          return (
            <div
              key={peerId}
              className="relative aspect-video bg-zinc-900 rounded border border-brand-border overflow-hidden flex items-center justify-center"
            >
              <video
                ref={(el) => {
                  if (el && remoteStream && el.srcObject !== remoteStream) {
                    el.srcObject = remoteStream;
                    remoteVideoRefs.current.set(peerId, el);
                  }
                }}
                autoPlay
                playsInline
                className="w-full h-full object-cover"
              />
              <div className="absolute bottom-1 left-1 px-1.5 py-0.5 rounded bg-black/70 text-[9px] font-mono text-zinc-300 truncate max-w-[80px]">
                {peerId.slice(0, 6)}
              </div>
            </div>
          );
        })}

        {peers.length === 0 && (
          <div className="relative aspect-video rounded border border-dashed border-zinc-800 flex flex-col items-center justify-center p-2 text-center">
            <span className="text-[10px] font-mono text-zinc-500">
              Waiting for others to join...
            </span>
          </div>
        )}
      </div>

      {/* Control bar */}
      <div className="p-2.5 bg-brand-surface border-t border-brand-border flex items-center justify-center gap-2">
        <button
          onClick={() => setMuted(!isMuted)}
          className={`p-2 rounded transition-colors ${
            isMuted ? 'bg-red-500/20 text-red-400 border border-red-500/30' : 'bg-brand-card text-brand-textHeading hover:bg-zinc-800 border border-brand-border'
          }`}
          title={isMuted ? 'Unmute Mic' : 'Mute Mic'}
        >
          {isMuted ? <MicOff className="w-4 h-4" /> : <Mic className="w-4 h-4" />}
        </button>

        <button
          onClick={() => setCameraOff(!isCameraOff)}
          className={`p-2 rounded transition-colors ${
            isCameraOff ? 'bg-red-500/20 text-red-400 border border-red-500/30' : 'bg-brand-card text-brand-textHeading hover:bg-zinc-800 border border-brand-border'
          }`}
          title={isCameraOff ? 'Turn Video On' : 'Turn Video Off'}
        >
          {isCameraOff ? <VideoOff className="w-4 h-4" /> : <Video className="w-4 h-4" />}
        </button>

        <button
          onClick={toggleScreenShare}
          className={`p-2 rounded transition-colors ${
            isScreenSharing ? 'bg-brand-amber text-brand-black' : 'bg-brand-card text-brand-textHeading hover:bg-zinc-800 border border-brand-border'
          }`}
          title={isScreenSharing ? 'Stop Screen Share' : 'Share Screen'}
        >
          <Monitor className="w-4 h-4" />
        </button>

        <button
          onClick={endCall}
          className="p-2 rounded bg-red-500 text-white hover:bg-red-600 transition-colors ml-2"
          title="Leave Call"
        >
          <PhoneOff className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
};
