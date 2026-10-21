import React from 'react';
import { useAuthStore } from '../../store/authStore';
import { LogOut } from 'lucide-react';

export const UserProfile: React.FC = () => {
  const { user, logout } = useAuthStore();

  if (!user) return null;

  const initials = user.name
    ? user.name
        .split(' ')
        .map((n) => n[0])
        .join('')
        .toUpperCase()
        .slice(0, 2)
    : 'U';

  return (
    <div className="p-3 border-t border-brand-border bg-brand-surface/40 flex items-center justify-between">
      <div className="flex items-center gap-2.5 overflow-hidden">
        <div className="w-7 h-7 rounded bg-brand-card border border-brand-border flex items-center justify-center font-mono text-xs font-semibold text-brand-amber shrink-0">
          {initials}
        </div>
        <div className="flex flex-col min-w-0">
          <span className="text-xs font-medium text-brand-textHeading truncate">{user.name}</span>
          <span className="text-[10px] font-mono text-brand-textMuted truncate">{user.email}</span>
        </div>
      </div>
      <button
        onClick={() => logout()}
        title="Sign Out"
        className="text-brand-textMuted hover:text-red-400 p-1.5 rounded hover:bg-brand-card transition-colors shrink-0"
      >
        <LogOut className="w-4 h-4" />
      </button>
    </div>
  );
};
