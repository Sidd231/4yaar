import React, { useEffect, useRef } from 'react';
import { ChatMessage } from '../../types/chat';
import { MessageBubble } from './MessageBubble';
import { Loader2 } from 'lucide-react';

interface MessageListProps {
  messages: ChatMessage[];
  isLoading: boolean;
  hasMore: boolean;
  onLoadOlder: () => void;
}

export const MessageList: React.FC<MessageListProps> = ({
  messages,
  isLoading,
  hasMore,
  onLoadOlder,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const prevScrollHeightRef = useRef<number>(0);
  const isPrependingRef = useRef<boolean>(false);

  // Auto-scroll to bottom on new incoming messages
  useEffect(() => {
    if (!isPrependingRef.current && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages.length]);

  // Preserve scroll position when older messages are prepended
  const handleScroll = () => {
    const el = containerRef.current;
    if (!el) return;

    if (el.scrollTop === 0 && hasMore && !isLoading) {
      isPrependingRef.current = true;
      prevScrollHeightRef.current = el.scrollHeight;
      onLoadOlder();
    }
  };

  useEffect(() => {
    if (isPrependingRef.current && containerRef.current) {
      const el = containerRef.current;
      const heightDiff = el.scrollHeight - prevScrollHeightRef.current;
      el.scrollTop = heightDiff;
      isPrependingRef.current = false;
    }
  }, [messages]);

  return (
    <div
      ref={containerRef}
      onScroll={handleScroll}
      className="flex-1 overflow-y-auto py-4 space-y-1"
    >
      {/* Loading older indicator */}
      {hasMore && (
        <div className="flex justify-center py-2">
          {isLoading ? (
            <Loader2 className="w-4 h-4 animate-spin text-brand-amber" />
          ) : (
            <button
              onClick={() => {
                isPrependingRef.current = true;
                if (containerRef.current) prevScrollHeightRef.current = containerRef.current.scrollHeight;
                onLoadOlder();
              }}
              className="text-[11px] font-mono text-zinc-500 hover:text-brand-textHeading transition-colors"
            >
              Load older messages
            </button>
          )}
        </div>
      )}

      {messages.length === 0 && !isLoading ? (
        <div className="h-full flex flex-col items-center justify-center text-center p-6 text-brand-textMuted space-y-2">
          <p className="font-mono text-xs text-zinc-500">
            This is the start of this channel's history.
          </p>
          <p className="text-xs">Say hello or share an update with the team.</p>
        </div>
      ) : (
        messages.map((msg) => <MessageBubble key={msg.id} message={msg} />)
      )}

      <div ref={bottomRef} />
    </div>
  );
};
