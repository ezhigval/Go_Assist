import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react-swc';
import { VitePWA } from 'vite-plugin-pwa';
import { resolve } from 'path';

// Platform detection based on environment variables
const isTelegram = process.env.VITE_PLATFORM === 'telegram';
const isWeb = process.env.VITE_PLATFORM === 'web';
const isMobile = process.env.VITE_PLATFORM === 'mobile';
const isDesktop = process.env.VITE_PLATFORM === 'desktop';

export default defineConfig({
  plugins: [
    react({
      // Enable React Fast Refresh
      fastRefresh: true,
      // Enable SWC minification
      devTarget: 'es2020',
    }),
    
    // PWA configuration for web platform
    ...(isWeb ? [{
      plugin: VitePWA({
        registerType: 'autoUpdate',
        includeAssets: ['favicon.ico', 'apple-touch-icon.png', 'masked-icon.svg'],
        manifest: {
          name: 'Modulr',
          short_name: 'Modulr',
          description: 'Cross-platform personal assistant',
          theme_color: '#000000',
          background_color: '#ffffff',
          display: 'standalone',
          orientation: 'portrait',
          scope: '/',
          start_url: '/',
          icons: [
            {
              src: 'pwa-192x192.png',
              sizes: '192x192',
              type: 'image/png',
            },
            {
              src: 'pwa-512x512.png',
              sizes: '512x512',
              type: 'image/png',
            },
          ],
        },
        workbox: {
          globPatterns: ['**/*.{js,css,html,ico,png,svg}'],
          runtimeCaching: [
            {
              urlPattern: /^https:\/\/api\.modulr\.app\/.*/i,
              handler: 'NetworkFirst',
              options: {
                cacheName: 'api-cache',
                expiration: {
                  maxEntries: 100,
                  maxAgeSeconds: 60 * 60 * 24, // 24 hours
                },
                cacheKeyWillBeUsed: async ({ request }) => {
                  return `${request.url}?modulr-cache=${Date.now()}`;
                },
              },
            },
          ],
        },
      }),
    }] : []),
  ],

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
      '@modulr/assets': resolve(__dirname, 'src/assets'),
    },
  },

  define: {
    // Platform-specific globals
    __PLATFORM__: JSON.stringify(process.env.VITE_PLATFORM || 'web'),
    __TELEGRAM__: isTelegram,
    __WEB__: isWeb,
    __MOBILE__: isMobile,
    __DESKTOP__: isDesktop,
    
    // Feature flags
    __DEV__: process.env.NODE_ENV === 'development',
    __ENABLE_ANALYTICS__: process.env.NODE_ENV === 'production',
  },

  build: {
    // Platform-specific build configurations
    ...(isTelegram && {
      rollupOptions: {
        input: {
          main: resolve(__dirname, 'src/telegram.main.tsx'),
        },
        output: {
          entryFileNames: 'telegram-[name].js',
          chunkFileNames: 'telegram-[name]-[hash].js',
          assetFileNames: 'telegram-[name]-[hash].[ext]',
        },
      },
    }),
    
    ...(isWeb && {
      rollupOptions: {
        output: {
          manualChunks: {
            vendor: ['react', 'react-dom'],
            router: ['react-router-dom'],
            ui: ['@radix-ui/react-dialog', '@radix-ui/react-dropdown-menu'],
            charts: ['recharts'],
            utils: ['date-fns', 'clsx', 'tailwind-merge'],
          },
        },
      },
    }),

    ...(isMobile && {
      rollupOptions: {
        input: {
          main: resolve(__dirname, 'src/mobile.main.tsx'),
        },
        output: {
          entryFileNames: 'mobile-[name].js',
          chunkFileNames: 'mobile-[name]-[hash].js',
          assetFileNames: 'mobile-[name]-[hash].[ext]',
        },
      },
    }),

    ...(isDesktop && {
      rollupOptions: {
        input: {
          main: resolve(__dirname, 'src/desktop.main.tsx'),
        },
        output: {
          entryFileNames: 'desktop-[name].js',
          chunkFileNames: 'desktop-[name]-[hash].js',
          assetFileNames: 'desktop-[name]-[hash].[ext]',
        },
      },
    }),

    // Common build settings
    target: 'es2020',
    minify: 'terser',
    sourcemap: true,
    cssCodeSplit: true,
    
    // Optimizations
    terserOptions: {
      compress: {
        drop_console: process.env.NODE_ENV === 'production',
        drop_debugger: true,
      },
    },
  },

  server: {
    port: 3000,
    host: true,
    
    // Proxy configuration for API calls
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
        ws: true, // Enable WebSocket proxy
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
        changeOrigin: true,
      },
    },
    
    // Headers for CORS and security
    headers: {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type, Authorization',
    },
  },

  preview: {
    port: 4173,
    host: true,
  },

  optimizeDeps: {
    include: [
      'react',
      'react-dom',
      'react-router-dom',
      '@tanstack/react-query',
      'zustand',
      'date-fns',
      'lucide-react',
    ],
    
    // Platform-specific optimizations
    ...(isTelegram && {
      exclude: ['@capacitor/geolocation', '@capacitor/camera'],
    }),
    
    ...(isMobile && {
      include: ['react-native-web'],
    }),
    
    ...(isDesktop && {
      include: ['@tauri-apps/api'],
    }),
  },

  css: {
    postcss: {
      plugins: [
        require('tailwindcss'),
        require('autoprefixer'),
      ],
    },
  },

  // Environment variables validation
  envPrefix: 'VITE_',
  
  // Test configuration
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    css: true,
  },
});
