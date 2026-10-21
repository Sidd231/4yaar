import React, { useState } from 'react';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { useChatStore } from '../../store/chatStore';
import { useNavigate } from 'react-router-dom';
import { X } from 'lucide-react';

interface CreateRoomDialogProps {
  isOpen: boolean;
  onClose: () => void;
}

export const CreateRoomDialog: React.FC<CreateRoomDialogProps> = ({ isOpen, onClose }) => {
  const [name, setName] = useState('');
  const [type, setType] = useState<'group' | 'direct'>('group');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createRoom = useChatStore((s) => s.createRoom);
  const navigate = useNavigate();

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    setLoading(true);
    setError(null);
    try {
      const room = await createRoom(name.trim().toLowerCase().replace(/\s+/g, '-'), type);
      onClose();
      setName('');
      navigate(`/chat/${room.id}`);
    } catch (err: any) {
      setError(err.response?.data?.error || err.message || 'Failed to create room');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-xs p-4">
      <div className="w-full max-w-md bg-brand-surface border border-brand-border rounded-lg shadow-2xl p-6 relative">
        <button
          onClick={onClose}
          className="absolute top-4 right-4 text-brand-textMuted hover:text-brand-textHeading transition-colors"
        >
          <X className="w-4 h-4" />
        </button>

        <h3 className="text-base font-semibold text-brand-textHeading font-display mb-1">
          Create Room
        </h3>
        <p className="text-xs text-brand-textMuted mb-5">
          Create a shared channel for real-time messaging and video calls.
        </p>

        <form onSubmit={handleSubmit} className="space-y-4">
          <Input
            id="room-name"
            label="Room Name"
            placeholder="e.g. dev-sync, design, general"
            value={name}
            onChange={(e) => setName(e.target.value)}
            error={error || undefined}
            autoFocus
            required
          />

          <div className="space-y-1.5">
            <label className="block text-xs font-mono text-brand-textMuted uppercase tracking-wider">
              Channel Type
            </label>
            <div className="grid grid-cols-2 gap-2">
              <button
                type="button"
                onClick={() => setType('group')}
                className={`px-3 py-2 text-xs font-medium rounded border transition-colors ${
                  type === 'group'
                    ? 'bg-brand-card border-brand-amber text-brand-textHeading'
                    : 'bg-brand-black/50 border-brand-border text-brand-textMuted hover:text-brand-textHeading'
                }`}
              >
                Group Room
              </button>
              <button
                type="button"
                onClick={() => setType('direct')}
                className={`px-3 py-2 text-xs font-medium rounded border transition-colors ${
                  type === 'direct'
                    ? 'bg-brand-card border-brand-amber text-brand-textHeading'
                    : 'bg-brand-black/50 border-brand-border text-brand-textMuted hover:text-brand-textHeading'
                }`}
              >
                Direct Room
              </button>
            </div>
          </div>

          <div className="flex justify-end gap-2.5 pt-3 border-t border-brand-border">
            <Button type="button" variant="ghost" size="sm" onClick={onClose} disabled={loading}>
              Cancel
            </Button>
            <Button type="submit" variant="primary" size="sm" isLoading={loading}>
              Create Channel
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
};
