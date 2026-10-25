import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../store/authStore';
import { Input } from '../components/ui/Input';
import { Button } from '../components/ui/Button';

export const LoginPage: React.FC = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const login = useAuthStore((s) => s.login);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email || !password) {
      setFormError('Email and password are required.');
      return;
    }

    setLoading(true);
    setFormError(null);
    try {
      await login(email.trim(), password);
      navigate('/chat');
    } catch (err: any) {
      setFormError(err.response?.data?.error || err.message || 'Authentication failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-brand-black flex flex-col justify-center items-center px-4">
      <div className="w-full max-w-sm border border-brand-border bg-brand-surface rounded-lg p-6 shadow-2xl">
        {/* Brand Header */}
        <div className="flex items-center gap-2.5 mb-6 justify-center">
          <img src="/4yaar.png" alt="4Yaar Logo" className="w-8 h-8 rounded-md object-contain" />
          <h1 className="font-display font-bold text-xl text-brand-textHeading tracking-tight flex items-center gap-1.5">
            4Yaar
            <span className="w-1.5 h-1.5 rounded-full bg-brand-amber" />
          </h1>
        </div>

        <div className="text-center mb-6">
          <h2 className="text-sm font-semibold text-brand-textHeading">Sign In to Workspace</h2>
          <p className="text-xs text-brand-textMuted mt-1">
            Private, invite-only team communication platform.
          </p>
        </div>

        {formError && (
          <div className="mb-4 p-2.5 rounded bg-red-500/10 border border-red-500/20 text-red-400 text-xs font-mono">
            {formError}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <Input
            id="login-email"
            label="Email Address"
            type="email"
            placeholder="name@example.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            autoFocus
          />

          <div className="relative">
            <Input
              id="login-password"
              label="Password"
              type={showPassword ? 'text' : 'password'}
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute right-2.5 top-7 text-[11px] font-mono text-brand-textMuted hover:text-brand-textHeading transition-colors"
            >
              {showPassword ? 'Hide' : 'Show'}
            </button>
          </div>

          <Button type="submit" variant="primary" className="w-full mt-2" isLoading={loading}>
            Sign In
          </Button>
        </form>

        <div className="mt-6 pt-4 border-t border-brand-border text-center">
          <p className="text-[11px] font-mono text-zinc-600">
            Registration requires an invite link from an administrator.
          </p>
        </div>
      </div>
    </div>
  );
};
