import { useEffect, useRef, useCallback } from 'react';
import { useAuthStore } from '../store/authStore';
import { useChatStore } from '../store/chatStore';
import { env } from '../lib/env';
import { ChatMessage, SendMessagePayload } from '../types/chat';

export function useChatSocket(roomId: string | null) {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttemptsRef = useRef(0);
  // Ref that tracks the roomId the *current* WS is connected to.
  // Lets onclose decide whether to reconnect to the same room or discard.
  const connectedRoomIdRef = useRef<string | null>(null);

  const accessToken = useAuthStore((s) => s.accessToken);
  const addIncomingMessage = useChatStore((s) => s.addIncomingMessage);
  const setWsConnected = useChatStore((s) => s.setWsConnected);

  const connect = useCallback(
    (targetRoomId: string) => {
      if (!accessToken) return;

      // Cancel any pending reconnect that was scheduled for a previous room
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }

      // Close the existing WS cleanly before opening a new one
      if (wsRef.current) {
        // Detach handlers so the old onclose doesn't schedule a reconnect
        wsRef.current.onclose = null;
        wsRef.current.onerror = null;
        wsRef.current.onmessage = null;
        wsRef.current.close();
        wsRef.current = null;
      }

      setWsConnected(false);
      connectedRoomIdRef.current = targetRoomId;
      reconnectAttemptsRef.current = 0;

      // Convert http/https base to ws/wss base if needed
      let base = env.wsBaseUrl;
      if (base.startsWith('http://')) base = base.replace('http://', 'ws://');
      if (base.startsWith('https://')) base = base.replace('https://', 'wss://');

      const wsUrl = `${base}/ws/chat/${targetRoomId}?token=${encodeURIComponent(accessToken)}`;
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        // Only mark connected if this is still the current room
        if (connectedRoomIdRef.current === targetRoomId) {
          setWsConnected(true);
          reconnectAttemptsRef.current = 0;
        }
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          // Expecting Message model schema from Go backend
          if (data && data.id && data.content !== undefined) {
            addIncomingMessage(data as ChatMessage);
          }
        } catch (err) {
          console.error('Failed to parse incoming WebSocket message', err);
        }
      };

      ws.onerror = () => {
        if (connectedRoomIdRef.current === targetRoomId) {
          setWsConnected(false);
        }
      };

      ws.onclose = () => {
        if (connectedRoomIdRef.current !== targetRoomId) {
          // This close event belongs to a stale connection for a different room — ignore it
          return;
        }

        setWsConnected(false);
        wsRef.current = null;

        // Auto-reconnect with exponential backoff (only for the current room)
        const nextAttempt = reconnectAttemptsRef.current + 1;
        reconnectAttemptsRef.current = nextAttempt;
        const delay = Math.min(1000 * Math.pow(1.5, nextAttempt) + Math.random() * 500, 15000);

        reconnectTimeoutRef.current = setTimeout(() => {
          // Double-check we are still on the same room before reconnecting
          if (connectedRoomIdRef.current === targetRoomId) {
            connect(targetRoomId);
          }
        }, delay);
      };
    },
    [accessToken, addIncomingMessage, setWsConnected]
  );

  useEffect(() => {
    if (!roomId) {
      // No room selected — tear down any existing connection
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      if (wsRef.current) {
        wsRef.current.onclose = null;
        wsRef.current.onerror = null;
        wsRef.current.onmessage = null;
        wsRef.current.close();
        wsRef.current = null;
      }
      connectedRoomIdRef.current = null;
      setWsConnected(false);
      return;
    }

    connect(roomId);

    return () => {
      // Switching rooms or unmounting — mark as stale so onclose won't reconnect
      connectedRoomIdRef.current = null;

      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      if (wsRef.current) {
        wsRef.current.onclose = null;
        wsRef.current.onerror = null;
        wsRef.current.onmessage = null;
        wsRef.current.close();
        wsRef.current = null;
      }
      setWsConnected(false);
    };
  }, [roomId, connect, setWsConnected]);

  const sendMessage = useCallback((payload: SendMessagePayload) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(payload));
      return true;
    }
    return false;
  }, []);

  return { sendMessage };
}
