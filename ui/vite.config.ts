import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    tsconfigPaths: true,
  },
  server: {
    port: 8008,
    proxy: {
      '/api': {
        target: 'http://192.168.150.28:8080',
        changeOrigin: true,
      },
    },
  },
});
