import React, { useState, useRef } from 'react';
import { Send, Paperclip, X, Loader2 } from 'lucide-react';
import { filesApi } from '../../api/files';
import { SendMessagePayload } from '../../types/chat';

interface MessageInputProps {
  roomId: string;
  onSendMessage: (payload: SendMessagePayload) => void;
  disabled?: boolean;
}

export const MessageInput: React.FC<MessageInputProps> = ({
  roomId,
  onSendMessage,
  disabled = false,
}) => {
  const [content, setContent] = useState('');
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState<number | null>(null);
  const [uploadError, setUploadError] = useState<string | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      const file = e.target.files[0];
      // Limit file size (50MB)
      if (file.size > 50 * 1024 * 1024) {
        setUploadError('File exceeds 50MB limit');
        return;
      }
      setSelectedFile(file);
      setUploadError(null);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (disabled || isUploading) return;
    if (!content.trim() && !selectedFile) return;

    let uploadedFileId: string | null = null;
    let messageType: 'text' | 'image' | 'file' = 'text';

    if (selectedFile) {
      setIsUploading(true);
      setUploadError(null);
      try {
        const uploadRes = await filesApi.uploadFile(selectedFile, roomId, (pct) => {
          setUploadProgress(pct);
        });
        uploadedFileId = (uploadRes as any)?.file?.id || uploadRes.id;
        messageType = selectedFile.type.startsWith('image/') ? 'image' : 'file';
      } catch (err: any) {
        setUploadError(err.response?.data?.error || 'Failed to upload attachment');
        setIsUploading(false);
        setUploadProgress(null);
        return;
      }
    }

    onSendMessage({
      content: content.trim() || (selectedFile ? selectedFile.name : ''),
      type: messageType,
      file_id: uploadedFileId,
    });

    setContent('');
    setSelectedFile(null);
    setIsUploading(false);
    setUploadProgress(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  return (
    <div className="p-3 border-t border-brand-border bg-brand-surface shrink-0">
      {/* File upload preview banner */}
      {selectedFile && (
        <div className="mb-2 p-2 rounded bg-brand-card border border-brand-border flex items-center justify-between text-xs font-mono">
          <div className="flex items-center gap-2 truncate">
            <Paperclip className="w-3.5 h-3.5 text-brand-amber shrink-0" />
            <span className="text-brand-textHeading truncate">{selectedFile.name}</span>
            {uploadProgress !== null && (
              <span className="text-zinc-500">({uploadProgress}%)</span>
            )}
          </div>
          <button
            type="button"
            onClick={() => {
              setSelectedFile(null);
              if (fileInputRef.current) fileInputRef.current.value = '';
            }}
            className="text-brand-textMuted hover:text-brand-textHeading p-1"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      )}

      {uploadError && (
        <div className="mb-2 text-xs text-red-400 font-mono">{uploadError}</div>
      )}

      <form onSubmit={handleSubmit} className="flex items-center gap-2">
        <input
          type="file"
          ref={fileInputRef}
          onChange={handleFileChange}
          className="hidden"
          id="file-upload-input"
        />

        <button
          type="button"
          onClick={() => fileInputRef.current?.click()}
          disabled={disabled || isUploading}
          className="p-2 text-brand-textMuted hover:text-brand-textHeading hover:bg-brand-card rounded transition-colors"
          title="Attach file"
        >
          <Paperclip className="w-4 h-4" />
        </button>

        <input
          type="text"
          value={content}
          onChange={(e) => setContent(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Type a message... (Press Enter to send)"
          disabled={disabled || isUploading}
          className="flex-1 bg-brand-black/60 border border-brand-border rounded px-3 py-2 text-xs text-brand-textHeading placeholder:text-zinc-600 focus:outline-none focus:border-brand-amber/70 transition-colors"
        />

        <button
          type="submit"
          disabled={disabled || isUploading || (!content.trim() && !selectedFile)}
          className="p-2 rounded bg-brand-amber text-brand-black hover:bg-amber-400 disabled:opacity-40 disabled:pointer-events-none transition-colors"
          title="Send message"
        >
          {isUploading ? (
            <Loader2 className="w-4 h-4 animate-spin" />
          ) : (
            <Send className="w-4 h-4" />
          )}
        </button>
      </form>
    </div>
  );
};
