<template>
  <div class="flex min-h-dvh flex-col bg-slate-50">
    <!-- Top bar; the safe-area padding keeps it clear of the notch in standalone mode. -->
    <header class="sticky top-0 z-20 bg-slate-900 pt-[env(safe-area-inset-top)] text-white shadow">
      <div class="mx-auto flex h-14 max-w-md items-center justify-between pr-1 pl-4">
        <RouterLink :to="{ name: 'track' }" class="flex items-center gap-2">
          <img src="../assets/logo.svg" alt="" class="size-8" />
          <span class="text-lg font-semibold tracking-tight">HomeGym</span>
        </RouterLink>
        <div class="flex items-center">
          <AppButton :to="{ name: 'dashboard' }" variant="inverse" size="touch" icon aria-label="Open the dashboard" title="Dashboard">
            <Squares2X2Icon class="size-6" aria-hidden="true" />
          </AppButton>
          <AppButton :to="{ name: 'track-settings' }" variant="inverse" size="touch" icon aria-label="Settings" title="Settings">
            <Cog6ToothIcon class="size-6" aria-hidden="true" />
          </AppButton>
          <AppButton variant="inverse" size="touch" icon aria-label="Sign out" title="Sign out" @click="logout">
            <ArrowRightStartOnRectangleIcon class="size-6" aria-hidden="true" />
          </AppButton>
        </div>
      </div>
    </header>

    <main class="mx-auto w-full max-w-md flex-1 px-4 pt-6 pb-[calc(1.5rem+env(safe-area-inset-bottom))]">
      <!-- Keyed by path so moving between two workouts mounts a fresh page. -->
      <RouterView :key="route.path" />
    </main>

    <!-- Tabs; hidden during a workout, whose own bar takes the bottom of the screen. -->
    <nav
      v-if="route.name !== 'workout'"
      class="sticky bottom-0 z-20 border-t border-slate-200 bg-white pb-[env(safe-area-inset-bottom)]"
      aria-label="Tracker"
    >
      <div class="mx-auto flex max-w-md">
        <RouterLink
          v-for="tab in tabs"
          :key="tab.name"
          :to="tab.to"
          class="flex min-h-14 flex-1 flex-col items-center justify-center gap-0.5 text-xs font-medium text-slate-500"
          exact-active-class="!text-indigo-600"
        >
          <component :is="tab.icon" class="size-6" aria-hidden="true" />
          {{ tab.name }}
        </RouterLink>
      </div>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import { ArrowRightStartOnRectangleIcon, CalendarDaysIcon, ClockIcon, Cog6ToothIcon, Squares2X2Icon } from '@heroicons/vue/24/outline'
import { logout } from '../api'
import AppButton from '../components/button/AppButton.vue'

const route = useRoute()

const tabs = [
  { name: 'This week', to: { name: 'track' }, icon: CalendarDaysIcon },
  { name: 'History', to: { name: 'history' }, icon: ClockIcon },
]
</script>
