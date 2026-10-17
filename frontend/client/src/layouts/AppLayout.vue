<template>
  <div class="flex min-h-dvh flex-col">
    <!-- Top bar; the safe-area padding keeps it clear of the notch in standalone mode. -->
    <header class="sticky top-0 z-10 bg-slate-900 pt-[env(safe-area-inset-top)] text-white shadow">
      <div class="mx-auto flex h-14 max-w-md items-center justify-between pr-1 pl-4">
        <div class="flex items-center gap-2">
          <img src="../assets/logo.svg" alt="" class="size-8" />
          <span class="text-lg font-semibold tracking-tight">HomeGym</span>
        </div>
        <button
          type="button"
          class="flex size-11 items-center justify-center rounded-xl text-slate-300 active:bg-slate-800"
          aria-label="Sign out"
          @click="logout"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="size-6" aria-hidden="true">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M15.75 9V5.25A2.25 2.25 0 0 0 13.5 3h-6a2.25 2.25 0 0 0-2.25 2.25v13.5A2.25 2.25 0 0 0 7.5 21h6a2.25 2.25 0 0 0 2.25-2.25V15m3 0 3-3m0 0-3-3m3 3H9"
            />
          </svg>
        </button>
      </div>
    </header>

    <main class="mx-auto w-full max-w-md flex-1 px-4 pt-6 pb-[calc(1.5rem+env(safe-area-inset-bottom))]">
      <RouterView />
    </main>

    <!-- A bottom tab bar goes here once there is more than one page. -->
  </div>
</template>

<script setup lang="ts">
import { logout } from '../api'
</script>
