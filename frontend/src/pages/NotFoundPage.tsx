import React from 'react';
import { NavLink } from 'react-router-dom';
import { Button } from '../components/ui/Button';

export const NotFoundPage: React.FC = () => {
  return (
    <div className="min-h-screen bg-brand-black flex flex-col justify-center items-center px-4 text-center">
      <span className="font-mono text-3xl font-bold text-brand-amber mb-2">404</span>
      <h1 className="text-base font-semibold text-brand-textHeading mb-2">Page Not Found</h1>
      <p className="text-xs text-brand-textMuted max-w-sm mb-6">
        The destination channel or link you requested does not exist or has been moved.
      </p>
      <NavLink to="/chat">
        <Button variant="secondary" size="sm">
          Return to Workspace
        </Button>
      </NavLink>
    </div>
  );
};
