import { createBrowserRouter, Navigate } from 'react-router-dom';
import { LoginPage } from '../pages/LoginPage';
import { RegisterPage } from '../pages/RegisterPage';
import { ChatPage } from '../pages/ChatPage';
import { AdminPage } from '../pages/AdminPage';
import { NotFoundPage } from '../pages/NotFoundPage';
import { ProtectedRoute, AdminRoute, PublicRoute } from './guards';

export const router = createBrowserRouter([
  // Public routes (login, register)
  {
    element: <PublicRoute />,
    children: [
      { path: '/login', element: <LoginPage /> },
      { path: '/register', element: <RegisterPage /> },
    ],
  },

  // Authenticated workspace routes
  {
    element: <ProtectedRoute />,
    children: [
      { path: '/', element: <Navigate to="/chat" replace /> },
      { path: '/chat', element: <ChatPage /> },
      { path: '/chat/:roomId', element: <ChatPage /> },
    ],
  },

  // Admin-only routes
  {
    element: <AdminRoute />,
    children: [{ path: '/admin', element: <AdminPage /> }],
  },

  // Fallback 404
  { path: '*', element: <NotFoundPage /> },
]);
