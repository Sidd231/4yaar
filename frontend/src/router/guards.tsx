import React from 'react';
import { Navigate, Outlet } from 'react-router-dom';
import { useAuthStore } from '../store/authStore';
import { Loader2 } from 'lucide-react';

export const ProtectedRoute: React.FC = () => {
  const { isAuthenticated, isInitializing } = useAuthStore();

  if (isInitializing) {
    return (
      <div className="min-h-screen bg-brand-black flex flex-col items-center justify-center font-mono text-xs text-zinc-500">
        <Loader2 className="w-5 h-5 animate-spin text-brand-amber mb-2" />
        <span>Restoring session...</span>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
};

export const AdminRoute: React.FC = () => {
  const { user, isAuthenticated, isInitializing } = useAuthStore();

  if (isInitializing) {
    return (
      <div className="min-h-screen bg-brand-black flex flex-col items-center justify-center font-mono text-xs text-zinc-500">
        <Loader2 className="w-5 h-5 animate-spin text-brand-amber mb-2" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (user?.role !== 'admin') {
    return <Navigate to="/chat" replace />;
  }

  return <Outlet />;
};

export const PublicRoute: React.FC = () => {
  const { isAuthenticated, isInitializing } = useAuthStore();

  if (isInitializing) {
    return null;
  }

  if (isAuthenticated) {
    return <Navigate to="/chat" replace />;
  }

  return <Outlet />;
};
