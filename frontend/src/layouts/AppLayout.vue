<template>
  <div class="min-h-screen bg-slate-50 text-slate-800">
    <!-- Desktop: fixed sidebar -->
    <div class="fixed inset-y-0 left-0 hidden w-64 lg:block">
      <SideNav :navigation="navigation" />
    </div>

    <!-- Mobile: top bar and slide-in drawer -->
    <header class="sticky top-0 z-30 flex h-14 items-center gap-3 border-b border-slate-200 bg-white px-4 lg:hidden">
      <button
        type="button"
        class="rounded-lg p-2 text-slate-600 hover:bg-slate-100"
        aria-label="Open navigation"
        :aria-expanded="drawerOpen"
        @click="drawerOpen = true"
      >
        <Bars3Icon class="size-6" aria-hidden="true" />
      </button>
      <span class="font-semibold text-slate-900">My Accessories</span>
    </header>

    <Transition
      enter-active-class="transition-opacity duration-200"
      leave-active-class="transition-opacity duration-150"
      enter-from-class="opacity-0"
      leave-to-class="opacity-0"
    >
      <div v-if="drawerOpen" class="fixed inset-0 z-40 bg-slate-950/60 lg:hidden" @click="drawerOpen = false" />
    </Transition>
    <Transition
      enter-active-class="transition-transform duration-200"
      leave-active-class="transition-transform duration-150"
      enter-from-class="-translate-x-full"
      leave-to-class="-translate-x-full"
    >
      <div v-if="drawerOpen" class="fixed inset-y-0 left-0 z-50 w-64 max-w-[85vw] lg:hidden">
        <SideNav :navigation="navigation" />
        <button
          type="button"
          class="absolute top-3 left-full ml-3 rounded-full bg-white/10 p-1.5 text-white hover:bg-white/20"
          aria-label="Close navigation"
          @click="drawerOpen = false"
        >
          <XMarkIcon class="size-5" aria-hidden="true" />
        </button>
      </div>
    </Transition>

    <main class="px-4 py-8 sm:px-6 lg:ml-64 lg:px-8">
      <!-- Keyed by path so moving between e.g. two trainings mounts a fresh page. -->
      <RouterView :key="route.path" />
    </main>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Bars3Icon, XMarkIcon } from '@heroicons/vue/24/outline'
import SideNav from '../components/navigation/SideNav.vue'
import { navigation } from '../navigation'

const drawerOpen = ref(false)

// Close the drawer after navigating and when Escape is pressed.
const route = useRoute()
watch(() => route.fullPath, () => (drawerOpen.value = false))

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') drawerOpen.value = false
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>
