<template>
  <aside class="flex h-full w-full flex-col bg-slate-900 text-slate-300">
    <div class="flex h-16 shrink-0 items-center gap-2 px-5">
      <span class="grid size-8 place-items-center rounded-lg bg-indigo-500 text-sm font-bold text-white">
        MA
      </span>
      <span class="font-semibold tracking-tight text-white">My Accessories</span>
    </div>

    <nav class="flex-1 overflow-y-auto px-3 py-4" aria-label="Main">
      <RouterLink
        v-for="item in navigation"
        :key="item.name"
        :to="item.to"
        class="mb-1 flex items-center gap-3 rounded-lg border-l-4 border-transparent px-3 py-2 text-sm font-medium transition-colors hover:bg-slate-800 hover:text-white"
        active-class="!border-indigo-400 bg-slate-800 !text-white"
      >
        <component :is="item.icon" class="size-5 shrink-0" aria-hidden="true" />
        {{ item.name }}
      </RouterLink>
    </nav>

    <div class="flex items-center gap-3 border-t border-slate-800 p-4">
      <span
        class="grid size-9 shrink-0 place-items-center rounded-full bg-slate-700 text-sm font-semibold text-white uppercase"
        aria-hidden="true"
      >
        {{ state.user?.email?.[0] }}
      </span>
      <div class="min-w-0 flex-1">
        <p class="truncate text-sm font-medium text-white">{{ state.user?.email }}</p>
        <button type="button" class="text-xs text-slate-400 hover:text-white hover:underline" @click="logout">
          Sign out
        </button>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { logout } from '../../api'
import { state } from '../../session'
import type { NavItem } from '../../navigation'

defineProps<{ navigation: NavItem[] }>()
</script>
