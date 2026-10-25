import React, { useEffect, useRef } from 'react';
import {
  Mic, MicOff, Video, VideoOff, Monitor, PhoneOff, Minimize2, Maximize2,
} from 'lucide-react';
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

  // ─── Minimized PIP ───────────────────────────────────────────────────────
  if (isMinimized) {
    return (
      <div className="fixed bottom-4 right-4 z-50 bg-brand-surface border border-brand-border rounded-lg shadow-2xl p-3 flex items-center gap-3 select-none animate-in slide-in-from-bottom-2 duration-200">
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

  // ─── Full-Screen Call View ────────────────────────────────────────────────
  const totalParticipants = peers.length + 1; // +1 for local
  const gridCols =
    totalParticipants <= 1 ? 'grid-cols-1' :
    totalParticipants <= 2 ? 'grid-cols-2' :
    totalParticipants <= 4 ? 'grid-cols-2' :
    'grid-cols-3';

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-zinc-950 select-none animate-in fade-in duration-200">
      {/* ── Header bar ── */}
      <div className="h-12 bg-brand-surface/80 backdrop-blur-sm border-b border-brand-border flex items-center justify-between px-5 shrink-0">
        <div className="flex items-center gap-2.5">
          <span className="w-2 h-2 rounded-full bg-brand-amber animate-pulse" />
          <span className="text-sm font-semibold text-brand-textHeading font-display">
            {activeCallRoomName}
          </span>
          <span className="text-xs font-mono text-zinc-500 ml-1">
            {status === 'connected'
              ? `${peers.length + 1} participant${peers.length + 1 !== 1 ? 's' : ''}`
              : status}
          </span>
        </div>

        <div className="flex items-center gap-1.5">
          {/* Minimize to PIP */}
          <button
            onClick={() => setMinimized(true)}
            className="p-1.5 text-brand-textMuted hover:text-brand-textHeading hover:bg-white/5 rounded transition-colors"
            title="Minimize"
          >
            <Minimize2 className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* ── Video Grid ── */}
      <div className={`flex-1 grid ${gridCols} gap-2 p-3 overflow-hidden`}>
        {/* Local tile */}
        <div className="relative bg-zinc-900 rounded-lg border border-white/10 overflow-hidden flex items-center justify-center">
          <video
            ref={localVideoRef}
            autoPlay
            playsInline
            muted
            className={`w-full h-full object-cover ${isCameraOff ? 'hidden' : ''}`}
          />
          {isCameraOff && (
            <div className="flex flex-col items-center gap-2 text-zinc-500">
              <VideoOff className="w-10 h-10" />
              <span className="text-xs font-mono">Camera Off</span>
            </div>
          )}
          <div className="absolute bottom-2 left-2 px-2 py-0.5 rounded bg-black/70 text-[11px] font-mono text-zinc-300">
            You {isMuted && '🔇'}
          </div>
          {/* Self-view label in top-right when in multi-user grid */}
          {peers.length > 0 && (
            <div className="absolute top-2 right-2 text-[10px] font-mono text-zinc-500">local</div>
          )}
        </div>

        {/* Remote participant tiles */}
        {peers.map((peerId) => {
          const remoteStream = getRemoteStream(peerId);
          return (
            <div
              key={peerId}
              className="relative bg-zinc-900 rounded-lg border border-white/10 overflow-hidden flex items-center justify-center"
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
              <div className="absolute bottom-2 left-2 px-2 py-0.5 rounded bg-black/70 text-[11px] font-mono text-zinc-300 truncate max-w-[140px]">
                {peerId.slice(0, 8)}…
              </div>
            </div>
          );
        })}

        {/* Waiting placeholder when alone */}
        {peers.length === 0 && (
          <div className="relative rounded-lg border border-dashed border-zinc-700 flex flex-col items-center justify-center gap-2 text-center p-4">
            <span className="text-3xl">👥</span>
            <span className="text-sm font-mono text-zinc-500">Waiting for others to join…</span>
          </div>
        )}
      </div>

      {/* ── Control Bar ── */}
      <div className="h-20 bg-brand-surface/80 backdrop-blur-sm border-t border-brand-border flex items-center justify-center gap-3 shrink-0 px-6">
        {/* Mic */}
        <button
          onClick={() => setMuted(!isMuted)}
          className={`p-3 rounded-full transition-colors ${
            isMuted
              ? 'bg-red-500/20 text-red-400 border border-red-500/30 hover:bg-red-500/30'
              : 'bg-brand-card text-brand-textHeading hover:bg-zinc-800 border border-brand-border'
          }`}
          title={isMuted ? 'Unmute Mic' : 'Mute Mic'}
        >
          {isMuted ? <MicOff className="w-5 h-5" /> : <Mic className="w-5 h-5" />}
        </button>

        {/* Camera */}
        <button
          onClick={() => setCameraOff(!isCameraOff)}
          className={`p-3 rounded-full transition-colors ${
            isCameraOff
              ? 'bg-red-500/20 text-red-400 border border-red-500/30 hover:bg-red-500/30'
              : 'bg-brand-card text-brand-textHeading hover:bg-zinc-800 border border-brand-border'
          }`}
          title={isCameraOff ? 'Turn Video On' : 'Turn Video Off'}
        >
          {isCameraOff ? <VideoOff className="w-5 h-5" /> : <Video className="w-5 h-5" />}
        </button>

        {/* Screen share */}
        <button
          onClick={toggleScreenShare}
          className={`p-3 rounded-full transition-colors ${
            isScreenSharing
              ? 'bg-brand-amber text-brand-black hover:bg-amber-400'
              : 'bg-brand-card text-brand-textHeading hover:bg-zinc-800 border border-brand-border'
          }`}
          title={isScreenSharing ? 'Stop Screen Share' : 'Share Screen'}
        >
          <Monitor className="w-5 h-5" />
        </button>

        {/* Divider */}
        <div className="w-px h-8 bg-brand-border mx-2" />

        {/* End call */}
        <button
          onClick={endCall}
          className="px-6 py-3 rounded-full bg-red-500 text-white hover:bg-red-600 transition-colors flex items-center gap-2 font-medium text-sm"
          title="Leave Call"
        >
          <PhoneOff className="w-4 h-4" />
          <span>Leave</span>
        </button>
      </div>
    </div>
  );
};
