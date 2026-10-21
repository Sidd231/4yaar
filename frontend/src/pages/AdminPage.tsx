import React, { useEffect, useState } from 'react';
import { NavLink } from 'react-router-dom';
import { invitesApi } from '../api/invites';
import { usersApi } from '../api/users';
import { UserListItem, Invite } from '../types/admin';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { formatDate } from '../lib/utils';
import { ArrowLeft, Mail, Search, Check, Copy } from 'lucide-react';

export const AdminPage: React.FC = () => {
  const [email, setEmail] = useState('');
  const [isSendingInvite, setIsSendingInvite] = useState(false);
  const [inviteResult, setInviteResult] = useState<Invite | null>(null);
  const [inviteError, setInviteError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const [users, setUsers] = useState<UserListItem[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [isLoadingUsers, setIsLoadingUsers] = useState(false);

  const loadUsers = async (query = '') => {
    setIsLoadingUsers(true);
    try {
      const data = await usersApi.listUsers(query);
      setUsers(data);
    } catch (e) {
      console.error('Failed to load users', e);
    } finally {
      setIsLoadingUsers(false);
    }
  };

  useEffect(() => {
    loadUsers();
  }, []);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    loadUsers(searchTerm);
  };

  const handleCreateInvite = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email.trim()) return;

    setIsSendingInvite(true);
    setInviteError(null);
    setInviteResult(null);

    try {
      const result = await invitesApi.create(email.trim());
      setInviteResult(result);
      setEmail('');
    } catch (err: any) {
      setInviteError(err.response?.data?.error || 'Failed to create invitation');
    } finally {
      setIsSendingInvite(false);
    }
  };

  const inviteLink = inviteResult
    ? `${window.location.origin}/register?token=${inviteResult.token}`
    : '';

  const handleCopyLink = () => {
    if (inviteLink) {
      navigator.clipboard.writeText(inviteLink);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="min-h-screen bg-brand-black text-brand-textBody font-sans">
      {/* Header */}
      <header className="h-14 border-b border-brand-border bg-brand-surface px-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <NavLink
            to="/chat"
            className="text-brand-textMuted hover:text-brand-textHeading p-1 rounded hover:bg-brand-card transition-colors"
            title="Back to Chat"
          >
            <ArrowLeft className="w-4 h-4" />
          </NavLink>
          <div className="flex items-center gap-2">
            <h1 className="text-sm font-semibold text-brand-textHeading">Admin Workspace Management</h1>
            <span className="text-[10px] font-mono uppercase px-1.5 py-0.5 rounded bg-brand-card border border-brand-border text-brand-amber">
              Admin Only
            </span>
          </div>
        </div>
      </header>

      <main className="max-w-4xl mx-auto p-6 space-y-8">
        {/* Create Invitation Card */}
        <section className="bg-brand-surface border border-brand-border rounded-lg p-5">
          <div className="flex items-center gap-2 mb-2">
            <Mail className="w-4 h-4 text-brand-amber" />
            <h2 className="text-sm font-semibold text-brand-textHeading">Invite New Team Member</h2>
          </div>
          <p className="text-xs text-brand-textMuted mb-4">
            Generate an invitation token. 4Yaar operates as a private, invite-only workspace.
          </p>

          <form onSubmit={handleCreateInvite} className="flex gap-2 max-w-md">
            <Input
              id="invite-email"
              type="email"
              placeholder="user@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
            <Button type="submit" variant="primary" size="md" isLoading={isSendingInvite} className="shrink-0">
              Send Invite
            </Button>
          </form>

          {inviteError && (
            <div className="mt-3 text-xs text-red-400 font-mono">{inviteError}</div>
          )}

          {inviteResult && (
            <div className="mt-4 p-3 rounded bg-brand-card border border-brand-border space-y-2">
              <div className="flex items-center justify-between text-xs">
                <span className="text-emerald-400 font-mono">✓ Invitation token created!</span>
                <span className="text-[10px] font-mono text-zinc-500">
                  Expires: {formatDate(inviteResult.expires_at)}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <input
                  readOnly
                  value={inviteLink}
                  className="flex-1 bg-brand-black/60 border border-brand-border rounded px-2.5 py-1 text-xs font-mono text-zinc-300"
                />
                <Button variant="secondary" size="sm" onClick={handleCopyLink}>
                  {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                  <span>{copied ? 'Copied' : 'Copy'}</span>
                </Button>
              </div>
            </div>
          )}
        </section>

        {/* User Directory Table */}
        <section className="bg-brand-surface border border-brand-border rounded-lg p-5">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-sm font-semibold text-brand-textHeading">Workspace Members</h2>
              <p className="text-xs text-brand-textMuted">
                Registered users authorized in this workspace.
              </p>
            </div>

            <form onSubmit={handleSearch} className="flex gap-2">
              <div className="relative">
                <Input
                  id="search-users"
                  placeholder="Filter name or email..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pr-8 text-xs h-8"
                />
                <button
                  type="submit"
                  className="absolute right-2 top-2 text-brand-textMuted hover:text-brand-textHeading"
                >
                  <Search className="w-4 h-4" />
                </button>
              </div>
            </form>
          </div>

          <div className="overflow-x-auto border border-brand-border rounded">
            <table className="w-full text-left text-xs">
              <thead className="bg-brand-card/60 text-brand-textMuted font-mono text-[10px] uppercase border-b border-brand-border">
                <tr>
                  <th className="px-4 py-2.5">Name</th>
                  <th className="px-4 py-2.5">Email</th>
                  <th className="px-4 py-2.5">Role</th>
                  <th className="px-4 py-2.5">Joined</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-brand-border/60">
                {isLoadingUsers ? (
                  <tr>
                    <td colSpan={4} className="px-4 py-6 text-center text-zinc-500 font-mono">
                      Loading users...
                    </td>
                  </tr>
                ) : users.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="px-4 py-6 text-center text-zinc-500 font-mono">
                      No users found.
                    </td>
                  </tr>
                ) : (
                  users.map((u) => (
                    <tr key={u.id} className="hover:bg-brand-card/30 transition-colors">
                      <td className="px-4 py-2.5 font-medium text-brand-textHeading">{u.name}</td>
                      <td className="px-4 py-2.5 font-mono text-zinc-400">{u.email}</td>
                      <td className="px-4 py-2.5">
                        <span
                          className={`font-mono text-[10px] uppercase px-1.5 py-0.5 rounded border ${
                            u.role === 'admin'
                              ? 'bg-brand-amber/10 border-brand-amber/30 text-brand-amber'
                              : 'bg-brand-card border-brand-border text-zinc-400'
                          }`}
                        >
                          {u.role}
                        </span>
                      </td>
                      <td className="px-4 py-2.5 font-mono text-zinc-500">{formatDate(u.created_at)}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>
      </main>
    </div>
  );
};
