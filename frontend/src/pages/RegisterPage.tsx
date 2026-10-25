import React, { useEffect, useState } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { invitesApi } from '../api/invites';
import { useAuthStore } from '../store/authStore';
import { Input } from '../components/ui/Input';
import { Button } from '../components/ui/Button';
import { Loader2, AlertCircle } from 'lucide-react';

export const RegisterPage: React.FC = () => {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token');
  const navigate = useNavigate();

  const registerWithInvite = useAuthStore((s) => s.registerWithInvite);

  const [isVerifying, setIsVerifying] = useState(true);
  const [inviteEmail, setInviteEmail] = useState<string | null>(null);
  const [verificationError, setVerificationError] = useState<string | null>(null);

  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  useEffect(() => {
    if (!token) {
      setIsVerifying(false);
      setVerificationError('Missing invitation token in URL query parameter (?token=...)');
      return;
    }

    async function verify() {
      try {
        const res = await invitesApi.verify(token!);
        if (res.valid) {
          setInviteEmail(res.email);
        } else {
          setVerificationError('This invitation token has expired or is invalid.');
        }
      } catch (err: any) {
        setVerificationError(
          err.response?.data?.error || 'Failed to verify invitation token. It may be expired or already used.'
        );
      } finally {
        setIsVerifying(false);
      }
    }

    verify();
  }, [token]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || password.length < 8) {
      setFormError('Please enter your full name and a password with at least 8 characters.');
      return;
    }

    setIsSubmitting(true);
    setFormError(null);
    try {
      await registerWithInvite(token!, name.trim(), password);
      navigate('/chat');
    } catch (err: any) {
      setFormError(err.response?.data?.error || err.message || 'Registration failed');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-brand-black flex flex-col justify-center items-center px-4">
      <div className="w-full max-w-sm border border-brand-border bg-brand-surface rounded-lg p-6 shadow-2xl">
        <div className="flex items-center gap-2.5 mb-6 justify-center">
          <img src="/4yaar.png" alt="4Yaar Logo" className="w-8 h-8 rounded-md object-contain" />
          <h1 className="font-display font-bold text-xl text-brand-textHeading tracking-tight flex items-center gap-1.5">
            4Yaar
            <span className="w-1.5 h-1.5 rounded-full bg-brand-amber" />
          </h1>
        </div>

        <div className="text-center mb-6">
          <h2 className="text-sm font-semibold text-brand-textHeading">Redeem Invitation</h2>
          <p className="text-xs text-brand-textMuted mt-1">
            Complete your profile to join the workspace.
          </p>
        </div>

        {isVerifying ? (
          <div className="py-8 flex flex-col items-center justify-center gap-2 text-zinc-500 font-mono text-xs">
            <Loader2 className="w-5 h-5 animate-spin text-brand-amber" />
            <span>Verifying invitation token...</span>
          </div>
        ) : verificationError ? (
          <div className="p-4 rounded bg-red-500/10 border border-red-500/20 text-center space-y-2">
            <AlertCircle className="w-6 h-6 text-red-400 mx-auto" />
            <p className="text-xs text-red-400 font-mono">{verificationError}</p>
            <Button variant="secondary" size="sm" onClick={() => navigate('/login')} className="mt-3">
              Back to Login
            </Button>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-xs font-mono text-brand-textMuted uppercase tracking-wider mb-1.5">
                Invited Email
              </label>
              <div className="w-full bg-brand-black/60 border border-brand-border rounded px-3 py-2 text-xs font-mono text-zinc-400 select-all">
                {inviteEmail}
              </div>
            </div>

            <Input
              id="register-name"
              label="Full Name"
              placeholder="e.g. Alex Rivera"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              autoFocus
            />

            <Input
              id="register-password"
              label="Password (min 8 chars)"
              type="password"
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />

            {formError && (
              <p className="text-xs text-red-400 font-mono">{formError}</p>
            )}

            <Button type="submit" variant="primary" className="w-full mt-2" isLoading={isSubmitting}>
              Join 4Yaar
            </Button>
          </form>
        )}
      </div>
    </div>
  );
};
