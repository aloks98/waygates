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
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: 'vendor-react',
              test: /node_modules\/(react|react-dom|scheduler)/,
            },
            {
              name: 'vendor-tanstack',
              test: /node_modules\/@tanstack/,
            },
            {
              name: 'vendor-ui',
              test: /node_modules\/(@e412\/titanium|lucide-react|sonner)/,
            },
          ],
        },
      },
    },
  },
});
