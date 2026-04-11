import { resolve } from 'node:path';
import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react-swc';
import { VitePWA } from 'vite-plugin-pwa';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), 'VITE_');
  const platform = env.VITE_PLATFORM || 'web';
  const isWeb = platform === 'web';

  const plugins = [react()];

  if (isWeb) {
    plugins.push(
      VitePWA({
        registerType: 'autoUpdate',
        includeAssets: ['favicon.ico'],
        manifest: {
          name: 'Modulr',
          short_name: 'Modulr',
          description: 'Cross-platform personal assistant',
          theme_color: '#0ea5e9',
          background_color: '#ffffff',
          display: 'standalone',
          start_url: '/',
          scope: '/',
          icons: [
            {
              src: '/pwa-192x192.png',
              sizes: '192x192',
              type: 'image/png',
            },
            {
              src: '/pwa-512x512.png',
              sizes: '512x512',
              type: 'image/png',
            },
          ],
        },
      })
    );
  }

  return {
    plugins,
    resolve: {
      alias: {
        '@': resolve(__dirname, 'src'),
        '@modulr/core-types': resolve(__dirname, 'src/types/core.ts'),
        '@modulr/ui': resolve(__dirname, 'src/components/ui'),
        '@modulr/hooks': resolve(__dirname, 'src/hooks'),
        '@modulr/lib': resolve(__dirname, 'src/lib'),
        '@modulr/context': resolve(__dirname, 'src/context'),
        '@modulr/platforms': resolve(__dirname, 'src/platforms'),
        '@modulr/modules': resolve(__dirname, 'src/modules'),
      },
    },
    define: {
      __PLATFORM__: JSON.stringify(platform),
      __TELEGRAM__: JSON.stringify(platform === 'telegram'),
      __WEB__: JSON.stringify(platform === 'web'),
      __MOBILE__: JSON.stringify(platform === 'mobile'),
      __DESKTOP__: JSON.stringify(platform === 'desktop'),
    },
    server: {
      port: 3000,
      host: true,
      proxy: {
        '/api': {
          target: env.VITE_API_BASE_URL || 'http://localhost:8080',
          changeOrigin: true,
          secure: false,
        },
        '/ws': {
          target: env.VITE_WS_URL || 'ws://localhost:8080',
          ws: true,
          changeOrigin: true,
        },
      },
    },
    preview: {
      port: 4173,
      host: true,
    },
    build: {
      target: 'es2020',
      sourcemap: true,
    },
    envPrefix: 'VITE_',
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: ['./src/test/setup.ts'],
      css: true,
    },
  };
});
