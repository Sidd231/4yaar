import React from 'react';
import { NavLink } from 'react-router-dom';
import { useAuthStore } from '../../store/authStore';
import { useChatStore } from '../../store/chatStore';
import { UserProfile } from './UserProfile';
import { Plus, Shield, Hash, User as UserIcon } from 'lucide-react';
import { cn } from '../../lib/utils';

interface SidebarProps {
  onOpenCreateRoom: () => void;
}

export const Sidebar: React.FC<SidebarProps> = ({ onOpenCreateRoom }) => {
  const { user } = useAuthStore();
  const { rooms, isLoadingRooms, unreadCounts } = useChatStore();

  const groupRooms = rooms.filter((r) => r.type !== 'direct');
  const directRooms = rooms.filter((r) => r.type === 'direct');

  return (
    <aside className="fixed left-0 top-0 h-full w-60 bg-brand-surface border-r border-brand-border z-40 flex flex-col justify-between select-none">
      <div className="flex flex-col h-[calc(100%-61px)]">
        {/* Brand Bar */}
        <div className="h-14 px-4 flex items-center justify-between border-b border-brand-border shrink-0">
          <div className="flex items-center gap-2.5">
            <img src="/4yaar.png" alt="4Yaar Logo" className="w-6 h-6 rounded object-contain" />
            <span className="font-display font-bold tracking-tight text-brand-textHeading text-[15px] flex items-center gap-1.5">
              4Yaar
              <span className="inline-block w-1.5 h-1.5 rounded-full bg-brand-amber" />
            </span>
          </div>
          {user?.role === 'admin' && (
            <NavLink
              to="/admin"
              className={({ isActive }) =>
                cn(
                  'text-brand-textMuted hover:text-brand-amber p-1 rounded hover:bg-brand-card transition-colors',
                  isActive && 'text-brand-amber'
                )
              }
              title="Admin Panel"
            >
              <Shield className="w-4 h-4" />
            </NavLink>
          )}
        </div>

        {/* Scrollable Navigation */}
        <div className="flex-1 px-3 py-4 space-y-6 overflow-y-auto">
          {/* Direct Messages Section */}
          <div>
            <div className="px-2 mb-2 font-mono text-[10px] tracking-widest uppercase text-brand-textMuted flex items-center justify-between">
              <span>Direct Messages</span>
              <span className="text-[9px] text-zinc-600 font-mono">
                {directRooms.length.toString().padStart(2, '0')}
              </span>
            </div>
            {directRooms.length === 0 ? (
              <p className="px-2 py-1 text-xs text-zinc-600 font-mono italic">No direct chats</p>
            ) : (
              <nav className="space-y-0.5">
                {directRooms.map((room) => {
                  const unread = unreadCounts[room.id] || 0;
                  return (
                    <NavLink
                      key={room.id}
                      to={`/chat/${room.id}`}
                      className={({ isActive }) =>
                        cn(
                          'group flex items-center justify-between px-2.5 py-1.5 rounded transition-colors text-xs font-normal',
                          isActive
                            ? 'bg-brand-card text-brand-textHeading font-medium border border-brand-border/60'
                            : 'text-brand-textBody hover:text-brand-textHeading hover:bg-brand-card'
                        )
                      }
                    >
                      <div className="flex items-center gap-2 truncate">
                        <UserIcon className="w-3.5 h-3.5 text-brand-textMuted group-hover:text-brand-amber transition-colors shrink-0" />
                        <span className="truncate">{room.name}</span>
                      </div>
                      {/* Unread badge or active indicator */}
                      {unread > 0 ? (
                        <span className="px-1.5 py-0.5 text-[10px] font-mono font-bold leading-none bg-brand-amber text-brand-black rounded-full shrink-0 animate-pulse shadow-sm">
                          {unread > 99 ? '99+' : unread}
                        </span>
                      ) : null}
                    </NavLink>
                  );
                })}
              </nav>
            )}
          </div>

          {/* Group Rooms Section */}
          <div>
            <div className="flex items-center justify-between px-2 mb-2 font-mono text-[10px] tracking-widest uppercase text-brand-textMuted">
              <span>Rooms</span>
              <button
                onClick={onOpenCreateRoom}
                className="text-brand-textMuted hover:text-brand-amber transition-colors p-0.5"
                title="Create Room"
                type="button"
              >
                <Plus className="w-3.5 h-3.5" />
              </button>
            </div>

            {isLoadingRooms ? (
              <div className="px-2 space-y-2">
                <div className="h-6 bg-brand-card/50 rounded animate-pulse" />
                <div className="h-6 bg-brand-card/50 rounded animate-pulse" />
              </div>
            ) : groupRooms.length === 0 ? (
              <div className="px-2 py-3 border border-dashed border-brand-border rounded text-center">
                <p className="text-xs text-brand-textMuted mb-2">No rooms created yet</p>
                <button
                  onClick={onOpenCreateRoom}
                  className="text-xs text-brand-amber hover:underline font-mono"
                >
                  + Create First Room
                </button>
              </div>
            ) : (
              <nav className="space-y-0.5">
                {groupRooms.map((room) => {
                  const unread = unreadCounts[room.id] || 0;
                  return (
                    <NavLink
                      key={room.id}
                      to={`/chat/${room.id}`}
                      className={({ isActive }) =>
                        cn(
                          'group flex items-center justify-between px-2.5 py-1.5 rounded transition-colors text-xs font-normal',
                          isActive
                            ? 'bg-brand-card text-brand-textHeading font-medium border border-brand-border/60'
                            : 'text-brand-textBody hover:text-brand-textHeading hover:bg-brand-card'
                        )
                      }
                    >
                      <div className="flex items-center gap-2 truncate">
                        <Hash className="w-3.5 h-3.5 text-brand-textMuted group-hover:text-brand-amber transition-colors shrink-0" />
                        <span className="truncate">{room.name}</span>
                      </div>
                      {/* Unread badge or active dot */}
                      {unread > 0 ? (
                        <span className="px-1.5 py-0.5 text-[10px] font-mono font-bold leading-none bg-brand-amber text-brand-black rounded-full shrink-0 animate-pulse shadow-sm">
                          {unread > 99 ? '99+' : unread}
                        </span>
                      ) : (
                        <span className="w-1.5 h-1.5 rounded-full bg-brand-amber shrink-0 opacity-0 group-[.active]:opacity-100" />
                      )}
                    </NavLink>
                  );
                })}
              </nav>
            )}
          </div>
        </div>
      </div>

      {/* User profile footer */}
      <UserProfile />
    </aside>
  );
};
