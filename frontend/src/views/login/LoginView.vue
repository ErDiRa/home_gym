<template>
	<div class="grid min-h-screen lg:grid-cols-2">
		<!-- Brand panel (large screens only) -->
		<aside
			class="hidden flex-col justify-between bg-slate-900 p-12 text-slate-300 lg:flex"
		>
			<div class="flex items-center gap-2">
				<img src="../../assets/logo.svg" alt="" class="size-9" />
				<span class="text-lg font-semibold text-white">HomeGym</span>
			</div>
			<div>
				<img src="../../assets/logo.svg" alt="" class="mb-8 size-28 rounded-3xl shadow-xl shadow-indigo-950/50" />
				<p class="text-3xl font-semibold tracking-tight text-white">
					Train smarter, move better.
				</p>
				<p class="mt-3 max-w-md text-slate-400">
					Your exercises, trainings and progress in one place.
				</p>
			</div>
			<p class="text-sm text-slate-500">&copy; HomeGym</p>
		</aside>

		<!-- Form -->
		<main class="flex items-center justify-center bg-slate-50 px-4 py-12">
			<div class="w-full max-w-sm">
				<!-- Brand for small screens, where the brand panel is hidden -->
				<div class="mb-8 flex items-center gap-3 lg:hidden">
					<img src="../../assets/logo.svg" alt="" class="size-16" />
					<span class="text-2xl font-semibold tracking-tight text-slate-900">HomeGym</span>
				</div>
				<h1 class="text-2xl font-bold tracking-tight text-slate-900">
					Welcome back
				</h1>
				<p class="mt-1 mb-8 text-sm text-slate-500">
					Sign in with your email and password.
				</p>

				<form class="space-y-5" @submit.prevent="submit">
					<label class="block text-sm font-medium text-slate-800">
						Email
						<input
							v-model="email"
							type="email"
							autocomplete="email"
							required
							class="mt-1 block h-10 w-full rounded-lg border border-slate-300 bg-white px-3 font-normal text-slate-900 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 focus:outline-none"
						/>
					</label>

					<label class="block text-sm font-medium text-slate-800">
						Password
						<input
							v-model="password"
							type="password"
							autocomplete="current-password"
							required
							class="mt-1 block h-10 w-full rounded-lg border border-slate-300 bg-white px-3 font-normal text-slate-900 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 focus:outline-none"
						/>
					</label>

					<label class="flex items-center gap-2 text-sm text-slate-600">
						<input
							v-model="remember"
							type="checkbox"
							class="size-4 rounded border-slate-300 accent-indigo-600"
						/>
						Keep me signed in
					</label>

					<p
						v-if="error"
						role="alert"
						class="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700"
					>
						{{ error }}
					</p>

					<AppButton type="submit" size="xl" class="w-full" :disabled="loading">
						{{ loading ? 'Signing in…' : 'Sign in' }}
					</AppButton>
				</form>
			</div>
		</main>
	</div>
</template>

<script setup lang="ts">
	import { ref } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { ApiError, login } from '../../api';
	import AppButton from '../../components/button/AppButton.vue';

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
