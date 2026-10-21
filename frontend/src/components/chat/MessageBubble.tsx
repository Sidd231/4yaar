import React from 'react';
import { ChatMessage } from '../../types/chat';
import { useAuthStore } from '../../store/authStore';
import { AttachmentCard } from './AttachmentCard';
import { formatTimestamp } from '../../lib/utils';
import { cn } from '../../lib/utils';

interface MessageBubbleProps {
  message: ChatMessage;
}

export const MessageBubble: React.FC<MessageBubbleProps> = ({ message }) => {
  const currentUser = useAuthStore((s) => s.user);
  const isMe = currentUser?.id === message.sender_id;

  const senderName = message.sender?.name || (isMe ? currentUser?.name : 'User') || 'Anonymous';
  const senderRole = message.sender?.role || (isMe ? currentUser?.role : '') || '';

  const initials = senderName
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);

  return (
    <div className={cn('group flex gap-3 px-4 py-2 hover:bg-brand-surface/30 transition-colors', isMe && 'bg-brand-surface/10')}>
      {/* Avatar / Monogram */}
      <div className="w-8 h-8 rounded bg-brand-card border border-brand-border flex items-center justify-center font-mono text-xs font-medium text-brand-textHeading shrink-0 mt-0.5">
        {initials}
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-baseline gap-2 mb-1">
          <span className="text-xs font-semibold text-brand-textHeading">{senderName}</span>
          {senderRole && (
            <span className="font-mono text-[9px] uppercase px-1.5 py-0.2 rounded bg-brand-card border border-brand-border text-zinc-400">
              {senderRole}
            </span>
          )}
          <span className="font-mono text-[10px] text-zinc-600">
            {formatTimestamp(message.created_at)}
          </span>
        </div>

        {/* Content */}
        {message.content && (
          <p className="text-xs text-brand-textHeading leading-relaxed break-words whitespace-pre-wrap select-text">
            {message.content}
          </p>
        )}

        {/* File attachment preview */}
        {message.file_id && (
          <AttachmentCard
            fileId={message.file_id}
            name={message.file?.name}
            size={message.file?.size}
            mimeType={message.file?.mime_type}
          />
        )}
      </div>
    </div>
  );
};
