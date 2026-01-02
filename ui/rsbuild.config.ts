import { defineConfig } from '@rsbuild/core';
import { pluginReact } from '@rsbuild/plugin-react';

export default defineConfig({
  plugins: [pluginReact()],
  html: {
    template: './index.html',
  },
  server: {
    port: 8008,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  source: {
    entry: {
      index: './src/main.tsx',
    },
  },
  tools: {
    postcss: {
      postcssOptions: {
        plugins: [require('@tailwindcss/postcss')],
      },
    },
  },
});
