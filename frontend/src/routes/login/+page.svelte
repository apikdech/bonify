<script lang="ts">
	import auth from '$lib/auth';
	import { Eye, EyeOff, Loader2 } from 'lucide-svelte';

	let email = '';
	let password = '';
	let rememberMe = false;
	let showPassword = false;
	let error = '';
	let isLoading = false;

	// Form validation
	let emailError = '';
	let passwordError = '';

	function validateEmail(email: string): boolean {
		const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
		return emailRegex.test(email);
	}

	function validateForm(): boolean {
		error = '';
		emailError = '';
		passwordError = '';

		let isValid = true;

		if (!email) {
			emailError = 'Email is required';
			isValid = false;
		} else if (!validateEmail(email)) {
			emailError = 'Please enter a valid email address';
			isValid = false;
		}

		if (!password) {
			passwordError = 'Password is required';
			isValid = false;
		}

		return isValid;
	}

	async function handleSubmit() {
		if (!validateForm()) return;

		isLoading = true;
		error = '';

		try {
			await auth.login(email, password);
			// On success, the auth store redirects to home page
		} catch (err: any) {
			isLoading = false;
			// Extract error message from API response
			error = err.message || 'Login failed. Please check your credentials and try again.';
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			handleSubmit();
		}
	}
</script>

<div class="min-h-screen bg-gray-50 flex flex-col items-center justify-center p-4">
	<!-- Card Container -->
	<div class="w-full max-w-md bg-white rounded-2xl shadow-xl p-8">
		<!-- Logo / Title -->
		<div class="text-center mb-8">
			<div class="w-16 h-16 bg-blue-600 rounded-xl flex items-center justify-center mx-auto mb-4 shadow-lg">
				<svg
					class="w-8 h-8 text-white"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
					/>
				</svg>
			</div>
			<h1 class="text-2xl font-bold text-gray-900">Receipt Manager</h1>
			<p class="text-gray-500 mt-1">Sign in to your account</p>
		</div>

		<!-- Error Banner -->
		{#if error}
			<div
				class="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg flex items-start gap-3"
				role="alert"
			>
				<svg class="w-5 h-5 text-red-500 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
					<path
						fill-rule="evenodd"
						d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
						clip-rule="evenodd"
					/>
				</svg>
				<div>
					<p class="text-sm font-medium text-red-800">Login Failed</p>
					<p class="text-sm text-red-600 mt-1">{error}</p>
				</div>
			</div>
		{/if}

		<!-- Login Form -->
		<form on:submit|preventDefault={handleSubmit} class="space-y-5">
			<!-- Email Field -->
			<div>
				<label for="email" class="block text-sm font-medium text-gray-700 mb-1">
					Email Address
				</label>
				<input
					type="email"
					id="email"
					name="email"
					bind:value={email}
					on:keydown={handleKeydown}
					class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-colors {emailError
						? 'border-red-500 focus:border-red-500 focus:ring-red-200'
						: ''}"
					placeholder="you@example.com"
					disabled={isLoading}
					required
				/>
				{#if emailError}
					<p class="mt-1 text-sm text-red-600">{emailError}</p>
				{/if}
			</div>

			<!-- Password Field -->
			<div>
				<label for="password" class="block text-sm font-medium text-gray-700 mb-1">
					Password
				</label>
				<div class="relative">
					<input
						type={showPassword ? 'text' : 'password'}
						id="password"
						name="password"
						bind:value={password}
						on:keydown={handleKeydown}
						class="w-full px-4 py-3 pr-12 rounded-lg border border-gray-300 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-colors {passwordError
							? 'border-red-500 focus:border-red-500 focus:ring-red-200'
							: ''}"
						placeholder="Enter your password"
						disabled={isLoading}
						required
					/>
					<button
						type="button"
						on:click={() => (showPassword = !showPassword)}
						class="absolute right-3 top-1/2 -translate-y-1/2 p-1 text-gray-400 hover:text-gray-600 transition-colors"
						tabindex="-1"
						aria-label={showPassword ? 'Hide password' : 'Show password'}
						disabled={isLoading}
					>
						{#if showPassword}
							<EyeOff class="w-5 h-5" />
						{:else}
							<Eye class="w-5 h-5" />
						{/if}
					</button>
				</div>
				{#if passwordError}
					<p class="mt-1 text-sm text-red-600">{passwordError}</p>
				{/if}
			</div>

			<!-- Remember Me -->
			<div class="flex items-center justify-between">
				<label class="flex items-center gap-2 cursor-pointer">
					<input
						type="checkbox"
						bind:checked={rememberMe}
						disabled={isLoading}
						class="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
					/>
					<span class="text-sm text-gray-600">Remember me</span>
				</label>
				<a
					href="#"
					class="text-sm font-medium text-blue-600 hover:text-blue-700 transition-colors"
					on:click|preventDefault={() => alert('Password reset coming soon!')}
				>
					Forgot password?
				</a>
			</div>

			<!-- Submit Button -->
			<button
				type="submit"
				disabled={isLoading}
				class="w-full py-3 px-4 bg-blue-600 hover:bg-blue-700 text-white font-semibold rounded-lg shadow-md hover:shadow-lg transform hover:-translate-y-0.5 active:translate-y-0 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none disabled:hover:shadow-md flex items-center justify-center gap-2"
			>
				{#if isLoading}
					<Loader2 class="w-5 h-5 animate-spin" />
					<span>Signing in...</span>
				{:else}
					<span>Sign In</span>
				{/if}
			</button>
		</form>

		<!-- Divider -->
		<div class="relative my-6">
			<div class="absolute inset-0 flex items-center">
				<div class="w-full border-t border-gray-200"></div>
			</div>
			<div class="relative flex justify-center text-sm">
				<span class="px-2 bg-white text-gray-400">New to Receipt Manager?</span>
			</div>
		</div>

		<!-- Sign Up Link -->
		<p class="text-center text-sm text-gray-600">
			Don't have an account?
			<a
				href="#"
				class="font-medium text-blue-600 hover:text-blue-700 transition-colors"
				on:click|preventDefault={() => alert('Registration coming soon!')}
			>
				Create account
			</a>
		</p>
	</div>

	<!-- Footer -->
	<p class="mt-8 text-sm text-gray-400 text-center">
		© 2024 Receipt Manager. All rights reserved.
	</p>
</div>
