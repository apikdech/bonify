<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import auth from '$lib/auth';
	import Navbar from '$components/Navbar.svelte';

	onMount(() => {
		// Check for PWA service worker
		if ('serviceWorker' in navigator) {
			navigator.serviceWorker.ready.then(() => {
				console.log('Service Worker ready');
			});
		}
	});

	// Route guards
	$: {
		const currentPath = $page.url.pathname;
		const isLoginPage = currentPath === '/login';
		const { isAuthenticated, isLoading } = $auth;

		// Wait for auth check to complete
		if (!isLoading) {
			// Redirect to /login if not authenticated and not on login page
			if (!isAuthenticated && !isLoginPage) {
				goto('/login');
			}
			// Redirect to / if authenticated and on login page
			if (isAuthenticated && isLoginPage) {
				goto('/');
			}
		}
	}
</script>

<div class="min-h-screen bg-white">
	<Navbar />
	<main class="md:ml-64">
		<slot />
	</main>
</div>
