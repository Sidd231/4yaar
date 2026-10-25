import { useEffect, useRef, useCallback } from 'react';
import { useAuthStore } from '../store/authStore';
import { useCallStore } from '../store/callStore';
import { env } from '../lib/env';
import { SignalMessage } from '../types/video';

// STUN ICE configuration
const defaultRtcConfig: RTCConfiguration = {
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' },
    { urls: 'stun:stun1.l.google.com:19302' },
  ],
};

export function useVideoCall() {
  const wsRef = useRef<WebSocket | null>(null);
  const localStreamRef = useRef<MediaStream | null>(null);
  const screenStreamRef = useRef<MediaStream | null>(null);
  const peerConnectionsRef = useRef<Map<string, RTCPeerConnection>>(new Map());
  const remoteStreamsRef = useRef<Map<string, MediaStream>>(new Map());
  const pendingIceCandidatesRef = useRef<Map<string, RTCIceCandidateInit[]>>(new Map());

  const user = useAuthStore((s) => s.user);
  const accessToken = useAuthStore((s) => s.accessToken);

  const {
    activeCallRoomId,
    status,
    isMuted,
    isCameraOff,
    isScreenSharing,
    setStatus,
    setPeers,
    addPeer,
    removePeer,
    setError,
    endCall,
  } = useCallStore();

  const sendSignal = useCallback((signal: SignalMessage) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(signal));
    }
  }, []);

  const createPeerConnection = useCallback((peerId: string) => {
    if (peerConnectionsRef.current.has(peerId)) {
      return peerConnectionsRef.current.get(peerId)!;
    }

    const pc = new RTCPeerConnection(defaultRtcConfig);
    peerConnectionsRef.current.set(peerId, pc);

    // Send local tracks
    if (localStreamRef.current) {
      localStreamRef.current.getTracks().forEach((track) => {
        pc.addTrack(track, localStreamRef.current!);
      });
    }

    pc.onicecandidate = (event) => {
      if (event.candidate && activeCallRoomId) {
        sendSignal({
          type: 'ice-candidate',
          room_id: activeCallRoomId,
          target_id: peerId,
          data: event.candidate.toJSON(),
        });
      }
    };

    pc.ontrack = (event) => {
      const stream = remoteStreamsRef.current.get(peerId) || new MediaStream();
      remoteStreamsRef.current.set(peerId, stream);
      event.streams[0]?.getTracks().forEach((t) => {
        if (!stream.getTracks().some((st) => st.id === t.id)) {
          stream.addTrack(t);
        }
      });
      addPeer(peerId);
    };

    pc.onconnectionstatechange = () => {
      if (pc.connectionState === 'failed' || pc.connectionState === 'closed') {
        pc.close();
        peerConnectionsRef.current.delete(peerId);
        remoteStreamsRef.current.delete(peerId);
        removePeer(peerId);
      }
    };

    return pc;
  }, [activeCallRoomId, sendSignal, addPeer, removePeer]);

  const initiateOffer = useCallback(async (targetPeerId: string) => {
    try {
      const pc = createPeerConnection(targetPeerId);
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);

      sendSignal({
        type: 'offer',
        room_id: activeCallRoomId!,
        target_id: targetPeerId,
        data: offer,
      });
    } catch (err: any) {
      console.error('Failed to create or send offer', err);
    }
  }, [createPeerConnection, sendSignal, activeCallRoomId]);

  const handleSignalMessage = useCallback(async (msg: SignalMessage) => {
    const senderId = msg.sender_id;
    if (!senderId || senderId === user?.id) return;

    switch (msg.type) {
      case 'existing-peers': {
        // Data contains an array of peer UUID strings
        const peerList: string[] = typeof msg.data === 'string' ? JSON.parse(msg.data) : (msg.data || []);
        setPeers(peerList);
        // We are the newcomer, so we send offers to all existing peers
        for (const peerId of peerList) {
          await initiateOffer(peerId);
        }
        break;
      }

      case 'peer-joined': {
        addPeer(senderId);
        break;
      }

      case 'peer-left': {
        removePeer(senderId);
        const pc = peerConnectionsRef.current.get(senderId);
        if (pc) {
          pc.close();
          peerConnectionsRef.current.delete(senderId);
        }
        remoteStreamsRef.current.delete(senderId);
        break;
      }

      case 'offer': {
        const pc = createPeerConnection(senderId);
        await pc.setRemoteDescription(new RTCSessionDescription(msg.data));

        // Flush buffered ICE candidates
        const buffered = pendingIceCandidatesRef.current.get(senderId) || [];
        for (const cand of buffered) {
          await pc.addIceCandidate(new RTCIceCandidate(cand));
        }
        pendingIceCandidatesRef.current.delete(senderId);

        const answer = await pc.createAnswer();
        await pc.setLocalDescription(answer);

        sendSignal({
          type: 'answer',
          room_id: activeCallRoomId!,
          target_id: senderId,
          data: answer,
        });
        break;
      }

      case 'answer': {
        const pc = peerConnectionsRef.current.get(senderId);
        if (pc) {
          await pc.setRemoteDescription(new RTCSessionDescription(msg.data));

          // Flush buffered ICE candidates
          const buffered = pendingIceCandidatesRef.current.get(senderId) || [];
          for (const cand of buffered) {
            await pc.addIceCandidate(new RTCIceCandidate(cand));
          }
          pendingIceCandidatesRef.current.delete(senderId);
        }
        break;
      }

      case 'ice-candidate': {
        const pc = peerConnectionsRef.current.get(senderId);
        if (pc && pc.remoteDescription && pc.remoteDescription.type) {
          try {
            await pc.addIceCandidate(new RTCIceCandidate(msg.data));
          } catch (e) {
            console.error('Error adding ICE candidate', e);
          }
        } else {
          // Buffer candidate until remote description is set
          const buffered = pendingIceCandidatesRef.current.get(senderId) || [];
          buffered.push(msg.data);
          pendingIceCandidatesRef.current.set(senderId, buffered);
        }
        break;
      }
    }
  }, [user?.id, setPeers, addPeer, removePeer, initiateOffer, createPeerConnection, sendSignal, activeCallRoomId]);

  // Connect WebRTC signaling and acquire media
  useEffect(() => {
    if (!activeCallRoomId || !accessToken) return;

    let isCancelled = false;

    async function initMediaAndSignaling() {
      try {
        setStatus('connecting');

        // Request audio and video with progressive fallback
        // NotReadableError happens when another tab/app holds the camera.
        // We try: ideal constraints → plain video:true → audio-only → empty stream.
        const tryGetMedia = async (): Promise<MediaStream> => {
          try {
            return await navigator.mediaDevices.getUserMedia({
              audio: true,
              video: { width: { ideal: 640 }, height: { ideal: 480 } },
            });
          } catch (e1: any) {
            console.warn('getUserMedia ideal failed, retrying with plain video:', e1);
            try {
              return await navigator.mediaDevices.getUserMedia({ audio: true, video: true });
            } catch (e2: any) {
              console.warn('getUserMedia with plain video failed, retrying audio-only:', e2);
              try {
                return await navigator.mediaDevices.getUserMedia({ audio: true });
              } catch (e3: any) {
                console.warn('getUserMedia audio-only failed, proceeding without media:', e3);
                return new MediaStream();
              }
            }
          }
        };
        localStreamRef.current = await tryGetMedia();

        if (isCancelled) return;

        // Establish WebSocket signaling connection
        let base = env.wsBaseUrl;
        if (base.startsWith('http://')) base = base.replace('http://', 'ws://');
        if (base.startsWith('https://')) base = base.replace('https://', 'wss://');

        const wsUrl = `${base}/ws/video/${activeCallRoomId}?token=${encodeURIComponent(accessToken!)}`;
        const ws = new WebSocket(wsUrl);
        wsRef.current = ws;

        ws.onopen = () => {
          if (!isCancelled) {
            setStatus('connected');
          }
        };

        ws.onmessage = (event) => {
          try {
            const signal: SignalMessage = JSON.parse(event.data);
            handleSignalMessage(signal);
          } catch (e) {
            console.error('Failed to parse signaling message', e);
          }
        };

        ws.onerror = () => {
          setError('Signaling connection error');
          setStatus('error');
        };

        ws.onclose = () => {
          if (!isCancelled) {
            setStatus('disconnected');
          }
        };
      } catch (err: any) {
        setError(err.message || 'Call initialization failed');
        setStatus('error');
      }
    }

    initMediaAndSignaling();

    return () => {
      isCancelled = true;

      // Close signaling
      if (wsRef.current) {
        wsRef.current.send(JSON.stringify({ type: 'leave', room_id: activeCallRoomId }));
        wsRef.current.close();
        wsRef.current = null;
      }

      // Stop local tracks
      if (localStreamRef.current) {
        localStreamRef.current.getTracks().forEach((track) => track.stop());
        localStreamRef.current = null;
      }

      if (screenStreamRef.current) {
        screenStreamRef.current.getTracks().forEach((track) => track.stop());
        screenStreamRef.current = null;
      }

      // Close all peer connections
      peerConnectionsRef.current.forEach((pc) => pc.close());
      peerConnectionsRef.current.clear();
      remoteStreamsRef.current.clear();
      pendingIceCandidatesRef.current.clear();
    };
  }, [activeCallRoomId, accessToken, setStatus, setError, handleSignalMessage]);

  // Audio track mute toggling
  useEffect(() => {
    if (localStreamRef.current) {
      localStreamRef.current.getAudioTracks().forEach((t) => {
        t.enabled = !isMuted;
      });
    }
  }, [isMuted]);

  // Video track mute toggling
  useEffect(() => {
    if (localStreamRef.current) {
      localStreamRef.current.getVideoTracks().forEach((t) => {
        t.enabled = !isCameraOff;
      });
    }
  }, [isCameraOff]);

  // Screen share handling
  const toggleScreenShare = useCallback(async () => {
    if (isScreenSharing) {
      // Revert to camera
      if (screenStreamRef.current) {
        screenStreamRef.current.getTracks().forEach((t) => t.stop());
        screenStreamRef.current = null;
      }
      const camTrack = localStreamRef.current?.getVideoTracks()[0];
      if (camTrack) {
        peerConnectionsRef.current.forEach((pc) => {
          const sender = pc.getSenders().find((s) => s.track?.kind === 'video');
          if (sender) sender.replaceTrack(camTrack);
        });
      }
      useCallStore.getState().setScreenSharing(false);
    } else {
      try {
        const stream = await navigator.mediaDevices.getDisplayMedia({ video: true });
        screenStreamRef.current = stream;
        const screenTrack = stream.getVideoTracks()[0];

        screenTrack.onended = () => {
          toggleScreenShare();
        };

        peerConnectionsRef.current.forEach((pc) => {
          const sender = pc.getSenders().find((s) => s.track?.kind === 'video');
          if (sender) sender.replaceTrack(screenTrack);
        });

        useCallStore.getState().setScreenSharing(true);
      } catch (e) {
        console.warn('Screen share canceled or failed', e);
      }
    }
  }, [isScreenSharing]);

  return {
    localStream: localStreamRef.current,
    getRemoteStream: (peerId: string) => remoteStreamsRef.current.get(peerId) || null,
    toggleScreenShare,
    endCall,
    status,
  };
}
