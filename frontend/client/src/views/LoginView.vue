<template>
  <main
    class="flex min-h-dvh flex-col justify-center px-6 pt-[calc(3rem+env(safe-area-inset-top))] pb-[calc(3rem+env(safe-area-inset-bottom))]"
  >
    <div class="mx-auto w-full max-w-sm">
      <div class="mb-10 flex flex-col items-center text-center">
        <img src="../assets/logo.svg" alt="" class="size-20 rounded-3xl shadow-lg shadow-indigo-950/20" />
        <h1 class="mt-5 text-2xl font-bold tracking-tight">HomeGym</h1>
        <p class="mt-1 text-sm text-slate-500">Sign in to see your training.</p>
      </div>

      <form class="space-y-5" @submit.prevent="submit">
        <!-- text-base (16px) keeps iOS from zooming into the focused input. -->
        <label class="block text-sm font-medium text-slate-800">
          Email
          <input
            v-model="email"
            type="email"
            inputmode="email"
            autocomplete="email"
            autocapitalize="none"
            required
            class="mt-1.5 block h-12 w-full rounded-xl border border-slate-300 bg-white px-4 text-base font-normal focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 focus:outline-none"
          />
        </label>

        <label class="block text-sm font-medium text-slate-800">
          Password
          <input
            v-model="password"
            type="password"
            autocomplete="current-password"
            required
            class="mt-1.5 block h-12 w-full rounded-xl border border-slate-300 bg-white px-4 text-base font-normal focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 focus:outline-none"
          />
        </label>

        <label class="flex min-h-11 items-center gap-3 text-sm text-slate-600">
          <input v-model="remember" type="checkbox" class="size-5 rounded border-slate-300 accent-indigo-600" />
          Keep me signed in
        </label>

        <p
          v-if="error"
          role="alert"
          class="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700"
        >
          {{ error }}
        </p>

        <AppButton type="submit" class="w-full" :disabled="loading">
          {{ loading ? 'Signing in…' : 'Sign in' }}
        </AppButton>
      </form>
    </div>
  </main>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiError, login } from '../api'
import AppButton from '../components/AppButton.vue'

const route = useRoute()
const router = useRouter()

const email = ref('')
const password = ref('')
const remember = ref(false)
const error = ref('')
const loading = ref(false)

// Only follow same-origin paths to avoid open redirects ("//host" would be another site).
function redirectTarget(): string {
  const target = route.query.redirect
  return typeof target === 'string' && target.startsWith('/') && !target.startsWith('//') ? target : '/'
}

function errorMessage(e: unknown): string {
  if (e instanceof ApiError && e.status === 401) return 'Invalid email or password.'
  return 'Could not sign in. Please try again.'
}

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await login(email.value, password.value, remember.value)
    // replace, so Back doesn't return to the login page.
    await router.replace(redirectTarget())
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    loading.value = false
  }
}
</script>
