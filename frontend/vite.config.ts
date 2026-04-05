import { defineConfig } from 'vite';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { VitePWA } from 'vite-plugin-pwa';

export default defineConfig({
	plugins: [
		sveltekit(),
		tailwindcss(),
		VitePWA({
			manifest: {
				name: 'Receipt Manager',
				short_name: 'Receipts',
				description: 'Track and manage your receipts',
				start_url: '/',
				scope: '/',
				display: 'standalone',
				theme_color: '#2563eb',
				background_color: '#ffffff',
				orientation: 'portrait',
				icons: [
					{
						src: '/icons/icon-192x192.png',
						sizes: '192x192',
						type: 'image/png'
					},
					{
						src: '/icons/icon-512x512.png',
						sizes: '512x512',
						type: 'image/png'
					}
				]
			},
			devOptions: {
				enabled: true,
				type: 'module'
			},
			workbox: {
				globPatterns: ['**/*.{js,css,html,png,svg,ico,woff,woff2}']
			}
		})
	]
});
