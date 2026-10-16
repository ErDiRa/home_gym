<template>
  <div class="flex grow flex-col gap-y-5 overflow-y-auto border-r border-slate-200 bg-white px-6">
    <div class="flex h-16 shrink-0 items-center">
      <span class="text-xl font-bold text-indigo-600">My Accessories</span>
    </div>
    <nav class="flex flex-1 flex-col">
      <ul role="list" class="flex flex-1 flex-col gap-y-7">
        <li>
          <ul role="list" class="-mx-2 space-y-1">
            <li v-for="item in navigation" :key="item.name">
              <RouterLink
                :to="item.to"
                class="group flex gap-x-3 rounded-md p-2 text-sm/6 font-semibold text-slate-700 hover:bg-slate-50 hover:text-indigo-600"
                active-class="bg-slate-50 !text-indigo-600"
              >
                <component
                  :is="item.icon"
                  class="size-6 shrink-0 text-slate-400 group-hover:text-indigo-600"
                  aria-hidden="true"
                />
                {{ item.name }}
              </RouterLink>
            </li>
          </ul>
        </li>
        <li class="-mx-6 mt-auto border-t border-slate-200">
          <div class="flex items-center justify-between gap-x-4 px-6 py-3 text-sm/6">
            <span class="truncate font-semibold text-slate-900">{{ state.user?.email }}</span>
            <button
              type="button"
              class="shrink-0 rounded-md px-2 py-1 font-semibold text-indigo-600 hover:bg-indigo-50"
              @click="logout"
            >
              Sign out
            </button>
          </div>
        </li>
      </ul>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { logout } from '../../api'
import { state } from '../../session'
import type { NavItem } from '../../navigation'

defineProps<{ navigation: NavItem[] }>()
</script>
