import { useEffect, useRef, useCallback } from 'react';
import { useAuthStore } from '../store/authStore';
import { useChatStore } from '../store/chatStore';
import { env } from '../lib/env';
import { ChatMessage, SendMessagePayload } from '../types/chat';

export function useChatSocket(roomId: string | null) {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const isDeliberateCloseRef = useRef(false);

  const accessToken = useAuthStore((s) => s.accessToken);
  const addIncomingMessage = useChatStore((s) => s.addIncomingMessage);
  const setWsConnected = useChatStore((s) => s.setWsConnected);

  const connect = useCallback(() => {
    if (!roomId || !accessToken) {
      return;
    }

    // Clean up previous connection if any
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    isDeliberateCloseRef.current = false;

    // Convert http/https base to ws/wss base if needed
    let base = env.wsBaseUrl;
    if (base.startsWith('http://')) base = base.replace('http://', 'ws://');
    if (base.startsWith('https://')) base = base.replace('https://', 'wss://');

    const wsUrl = `${base}/ws/chat/${roomId}?token=${encodeURIComponent(accessToken)}`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setWsConnected(true);
      reconnectAttemptsRef.current = 0;
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
      setWsConnected(false);
    };

    ws.onclose = () => {
      setWsConnected(false);
      wsRef.current = null;

      // Auto-reconnect with exponential backoff if not closed intentionally
      if (!isDeliberateCloseRef.current) {
        const nextAttempt = reconnectAttemptsRef.current + 1;
        reconnectAttemptsRef.current = nextAttempt;
        const delay = Math.min(1000 * Math.pow(1.5, nextAttempt) + Math.random() * 500, 15000);

        reconnectTimeoutRef.current = setTimeout(() => {
          connect();
        }, delay);
      }
    };
  }, [roomId, accessToken, addIncomingMessage, setWsConnected]);

  useEffect(() => {
    connect();

    return () => {
      isDeliberateCloseRef.current = true;
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      setWsConnected(false);
    };
  }, [connect, setWsConnected]);

  const sendMessage = useCallback((payload: SendMessagePayload) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(payload));
      return true;
    }
    return false;
  }, []);

  return { sendMessage };
}
