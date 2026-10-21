export const env = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1',
  wsBaseUrl: import.meta.env.VITE_WS_BASE_URL || 'ws://localhost:8080',
};
