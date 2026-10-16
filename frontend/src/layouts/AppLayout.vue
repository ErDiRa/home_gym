<template>
  <div class="min-h-screen bg-slate-50 text-slate-800">
    <!-- Mobile sidebar -->
    <div v-if="sidebarOpen" class="relative z-50 lg:hidden" role="dialog" aria-modal="true">
      <div class="fixed inset-0 bg-slate-900/80" @click="sidebarOpen = false" />
      <div class="fixed inset-y-0 left-0 flex w-72 max-w-full">
        <SideNav :navigation="navigation" />
        <button
          type="button"
          class="absolute top-4 left-full ml-4 text-white"
          @click="sidebarOpen = false"
        >
          <span class="sr-only">Close sidebar</span>
          <XMarkIcon class="size-6" aria-hidden="true" />
        </button>
      </div>
    </div>

    <!-- Desktop sidebar -->
    <div class="hidden lg:fixed lg:inset-y-0 lg:z-40 lg:flex lg:w-72 lg:flex-col">
      <SideNav :navigation="navigation" />
    </div>

    <!-- Mobile top bar -->
    <div
      class="sticky top-0 z-30 flex items-center gap-x-4 bg-white px-4 py-4 shadow-sm lg:hidden"
    >
      <button type="button" class="-m-2.5 p-2.5 text-slate-700" @click="sidebarOpen = true">
        <span class="sr-only">Open sidebar</span>
        <Bars3Icon class="size-6" aria-hidden="true" />
      </button>
      <span class="text-sm/6 font-semibold text-slate-900">My Accessories</span>
    </div>

    <main class="lg:pl-72">
      <div class="px-4 py-8 sm:px-6 lg:px-8">
        <RouterView />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Bars3Icon, XMarkIcon } from '@heroicons/vue/24/outline'
import SideNav from '../components/navigation/SideNav.vue'
import { navigation } from '../navigation'

const sidebarOpen = ref(false)

// Close the mobile sidebar after navigating.
const route = useRoute()
watch(() => route.fullPath, () => (sidebarOpen.value = false))
</script>
