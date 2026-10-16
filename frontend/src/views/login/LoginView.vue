<template>
	<div
		class="flex min-h-screen items-center justify-center bg-slate-50 px-4 py-12"
	>
		<div
			class="w-full max-w-md rounded-xl bg-white px-8 py-10 shadow-sm sm:px-12"
		>
			<h1 class="mb-8 text-center text-2xl font-bold text-slate-900">
				Sign in to your account
			</h1>

			<form class="space-y-6" @submit.prevent="submit">
				<div>
					<label for="email" class="block text-sm font-medium text-slate-900"
						>Email address</label
					>
					<input
						id="email"
						v-model="email"
						type="email"
						autocomplete="email"
						required
						class="mt-2 block w-full rounded-md border border-slate-300 px-3 py-2.5 text-slate-900 outline-none focus:border-indigo-600 focus:ring-2 focus:ring-indigo-200"
					/>
				</div>

				<div>
					<label for="password" class="block text-sm font-medium text-slate-900"
						>Password</label
					>
					<input
						id="password"
						v-model="password"
						type="password"
						autocomplete="current-password"
						required
						class="mt-2 block w-full rounded-md border border-slate-300 px-3 py-2.5 text-slate-900 outline-none focus:border-indigo-600 focus:ring-2 focus:ring-indigo-200"
					/>
				</div>

				<p
					v-if="error"
					role="alert"
					class="rounded-md bg-rose-50 px-3 py-2 text-sm text-rose-700"
				>
					{{ error }}
				</p>

				<button
					type="submit"
					:disabled="loading"
					class="w-full rounded-md bg-indigo-600 px-3 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
				>
					{{ loading ? 'Signing in…' : 'Sign in' }}
				</button>
			</form>
		</div>
	</div>
</template>

<script setup lang="ts">
	import { ref } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { ApiError, login } from '../../api';

	const route = useRoute();
	const router = useRouter();

	const email = ref('');
	const password = ref('');
	const remember = ref(false);
	const error = ref('');
	const loading = ref(false);

	// Only follow same-origin paths to avoid open redirects.
	function redirectTarget(): string {
		const target = route.query.redirect;
		return typeof target === 'string' &&
			target.startsWith('/') &&
			!target.startsWith('//')
			? target
			: '/';
	}

	async function submit() {
		error.value = '';
		loading.value = true;
		try {
			await login(email.value, password.value, remember.value);
			await router.replace(redirectTarget());
		} catch (e) {
			error.value =
				e instanceof ApiError && e.status === 401
					? 'Invalid email or password.'
					: 'Could not sign in. Please try again.';
		} finally {
			loading.value = false;
		}
	}
</script>
