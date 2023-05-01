import react from '@vitejs/plugin-react'
import jotaiDebugLabel from 'jotai/babel/plugin-debug-label'
import jotaiReactRefresh from 'jotai/babel/plugin-react-refresh'
import UnoCSS from 'unocss/vite'
import { defineConfig, splitVendorChunkPlugin } from 'vite'
import { VitePWA } from 'vite-plugin-pwa'
import tsConfigPath from 'vite-tsconfig-paths'

export default defineConfig(
    env => ({
        plugins: [
            // only use react-fresh
            env.mode === 'development' && react({
                babel: { plugins: [jotaiDebugLabel, jotaiReactRefresh] },
            }),
            tsConfigPath(),
            UnoCSS(),
            VitePWA({
                injectRegister: 'inline',
                manifest: {
                    start_url: '/',
                    short_name: 'LightSocks Dashboard',
                    name: 'LightSocks Dashboard',
                },
            }),
            splitVendorChunkPlugin(),
        ],
        server: {
            port: 3000,
            proxy: {
                '/api': {
                    target: 'http://localhost:8080/api',
                    changeOrigin: true,
                },
            },
        },
        base: './',
        css: {
            preprocessorOptions: {
                scss: {
                    additionalData: '@use "sass:math"; @import "src/styles/variables.scss";',
                },
            },
        },
        build: {
            outDir: '../static',
            reportCompressedSize: false,
            emptyOutDir: true,
        },
        esbuild: {
            jsxInject: "import React from 'react'",
        },
    }),
)
