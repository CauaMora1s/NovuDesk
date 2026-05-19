import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		host: '0.0.0.0',
		port: 5173,
		watch: {
			usePolling: true,
			interval: 1000
		},
		proxy: {
			'/api': {
				target: 'http://api:8080',
				changeOrigin: true,
				proxyTimeout: 10000,
				timeout: 10000
			}
		}
	},
	test: {
		include: ['src/**/*.{test,spec}.{js,ts}'],
		environment: 'jsdom'
	}
});
