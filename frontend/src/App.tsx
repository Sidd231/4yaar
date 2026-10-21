
import React, { useEffect } from 'react';
import { RouterProvider } from 'react-router-dom';

import { router } from './router';
import { useAuthStore } from './store/authStore';

export const App: React.FC = () => {
  const initAuth = useAuthStore((state) => state.initAuth);
  const isInitializing = useAuthStore(
    (state) => state.isInitializing
  );

  useEffect(() => {
    void initAuth();
  }, [initAuth]);

  if (isInitializing) {
    return (
      <div
        style={{
          minHeight: '100vh',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        Loading your session...
      </div>
    );
  }

  return <RouterProvider router={router} />;
};

export default App;
