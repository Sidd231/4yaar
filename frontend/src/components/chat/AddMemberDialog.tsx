import React, { useState, useEffect } from 'react';
import { Button } from '../ui/Button';
import { usersApi } from '../../api/users';
import { chatApi } from '../../api/chat';
import { UserListItem } from '../../types/admin';
import { Room } from '../../types/chat';
import { useChatStore } from '../../store/chatStore';
import { X, UserPlus, Search, Check, AlertCircle, Loader2 } from 'lucide-react';

interface AddMemberDialogProps {
  isOpen: boolean;
  onClose: () => void;
  room: Room;
}

export const AddMemberDialog: React.FC<AddMemberDialogProps> = ({
  isOpen,
  onClose,
  room,
}) => {
  const [users, setUsers] = useState<UserListItem[]>([]);
  const [search, setSearch] = useState('');
  const [isLoadingUsers, setIsLoadingUsers] = useState(false);
  const [addingUserId, setAddingUserId] = useState<string | null>(null);
  const [successUserId, setSuccessUserId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [customUserId, setCustomUserId] = useState('');

  const fetchRooms = useChatStore((s) => s.fetchRooms);

  // Existing member IDs in the room
  const memberIds = new Set((room.members || []).map((m) => m.user_id));

  useEffect(() => {
    if (!isOpen) return;
    let isCancelled = false;

    const loadUsers = async () => {
      setIsLoadingUsers(true);
      setError(null);
      try {
        const data = await usersApi.listUsers(search);
        if (!isCancelled) {
          setUsers(data);
        }
      } catch (err: any) {
        if (!isCancelled) {
          setError(err.response?.data?.error || 'Failed to search users');
        }
      } finally {
        if (!isCancelled) {
          setIsLoadingUsers(false);
        }
      }
    };

    const timer = setTimeout(() => {
      loadUsers();
    }, 200);

    return () => {
      isCancelled = true;
      clearTimeout(timer);
    };
  }, [isOpen, search]);

  if (!isOpen) return null;

  const handleAddMember = async (userId: string) => {
    setAddingUserId(userId);
    setError(null);
    try {
      await chatApi.addMember(room.id, userId);
      setSuccessUserId(userId);
      await fetchRooms();
      setTimeout(() => {
        setSuccessUserId(null);
      }, 2000);
    } catch (err: any) {
      setError(
        err.response?.data?.error || err.message || 'Failed to add member to room'
      );
    } finally {
      setAddingUserId(null);
    }
  };

  const handleAddCustomUser = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!customUserId.trim()) return;
    await handleAddMember(customUserId.trim());
    setCustomUserId('');
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-xs p-4 animate-in fade-in duration-150">
      <div className="w-full max-w-md bg-brand-surface border border-brand-border rounded-lg shadow-2xl p-6 relative flex flex-col max-h-[85vh]">
        {/* Close Button */}
        <button
          onClick={onClose}
          className="absolute top-4 right-4 text-brand-textMuted hover:text-brand-textHeading transition-colors p-1"
          aria-label="Close"
        >
          <X className="w-4 h-4" />
        </button>

        {/* Header */}
        <div className="flex items-center gap-2.5 mb-1">
          <div className="w-7 h-7 rounded bg-brand-amber/10 border border-brand-amber/30 flex items-center justify-center text-brand-amber">
            <UserPlus className="w-4 h-4" />
          </div>
          <h3 className="text-base font-semibold text-brand-textHeading font-display">
            Add Members to #{room.name}
          </h3>
        </div>
        <p className="text-xs text-brand-textMuted mb-4">
          Invite teammates or enter an existing user UUID to join this room.
        </p>

        {/* Error Alert */}
        {error && (
          <div className="mb-4 p-2.5 rounded bg-red-500/10 border border-red-500/20 text-red-400 text-xs flex items-center gap-2">
            <AlertCircle className="w-4 h-4 shrink-0" />
            <span className="flex-1">{error}</span>
          </div>
        )}

        {/* Search registered users */}
        <div className="relative mb-3">
          <Search className="w-4 h-4 absolute left-3 top-2.5 text-brand-textMuted" />
          <input
            type="text"
            placeholder="Search users by name or email..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full bg-brand-black/60 border border-brand-border rounded pl-9 pr-3 py-2 text-xs text-brand-textHeading placeholder:text-zinc-600 focus:outline-none focus:border-brand-amber/80 focus:ring-1 focus:ring-brand-amber/40 transition-colors"
          />
        </div>

        {/* User Search Results List */}
        <div className="flex-1 overflow-y-auto space-y-1.5 min-h-[140px] max-h-[220px] pr-1">
          {isLoadingUsers ? (
            <div className="flex items-center justify-center h-28 text-brand-textMuted gap-2 text-xs font-mono">
              <Loader2 className="w-4 h-4 animate-spin text-brand-amber" />
              <span>Searching users...</span>
            </div>
          ) : users.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-28 text-center text-brand-textMuted text-xs px-4">
              <p>No registered users found matching your search.</p>
            </div>
          ) : (
            users.map((u) => {
              const isAlreadyMember = memberIds.has(u.id);
              const isAdding = addingUserId === u.id;
              const isSuccess = successUserId === u.id;

              return (
                <div
                  key={u.id}
                  className="flex items-center justify-between p-2.5 rounded border border-brand-border/60 bg-brand-black/40 hover:bg-brand-black/80 transition-colors"
                >
                  <div className="min-w-0 pr-3">
                    <p className="text-xs font-medium text-brand-textHeading truncate">
                      {u.name}
                    </p>
                    <p className="text-[11px] text-brand-textMuted font-mono truncate">
                      {u.email}
                    </p>
                  </div>
                  <div>
                    {isAlreadyMember || isSuccess ? (
                      <span className="inline-flex items-center gap-1 text-[11px] font-mono text-emerald-400 bg-emerald-500/10 px-2 py-1 rounded border border-emerald-500/20">
                        <Check className="w-3 h-3" />
                        <span>Member</span>
                      </span>
                    ) : (
                      <Button
                        size="sm"
                        variant="secondary"
                        isLoading={isAdding}
                        onClick={() => handleAddMember(u.id)}
                        className="text-xs px-2.5 py-1 h-7 border-brand-border/80 hover:border-brand-amber/50 hover:text-brand-amber"
                      >
                        Add
                      </Button>
                    )}
                  </div>
                </div>
              );
            })
          )}
        </div>

        {/* Direct UUID Input fallback */}
        <div className="mt-4 pt-3 border-t border-brand-border">
          <form onSubmit={handleAddCustomUser} className="space-y-1.5">
            <label className="block text-[11px] font-mono text-brand-textMuted uppercase tracking-wider">
              Or Add by User ID (UUID)
            </label>
            <div className="flex gap-2">
              <input
                type="text"
                placeholder="e.g. 550e8400-e29b-41d4-a716-446655440000"
                value={customUserId}
                onChange={(e) => setCustomUserId(e.target.value)}
                className="flex-1 bg-brand-black/60 border border-brand-border rounded px-3 py-1.5 text-xs text-brand-textHeading placeholder:text-zinc-600 focus:outline-none focus:border-brand-amber/80 font-mono transition-colors"
              />
              <Button
                type="submit"
                variant="primary"
                size="sm"
                disabled={!customUserId.trim() || addingUserId === customUserId.trim()}
                isLoading={addingUserId === customUserId.trim()}
              >
                Add
              </Button>
            </div>
          </form>
        </div>

        {/* Modal Actions */}
        <div className="flex justify-end pt-4 mt-2 border-t border-brand-border">
          <Button variant="ghost" size="sm" onClick={onClose}>
            Done
          </Button>
        </div>
      </div>
    </div>
  );
};
