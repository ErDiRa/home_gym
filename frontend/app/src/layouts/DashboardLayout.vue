<template>
  <div class="min-h-screen bg-slate-50 text-slate-800">
    <!-- Desktop: fixed sidebar -->
    <div class="fixed inset-y-0 left-0 hidden w-64 lg:block">
      <SideNav :navigation="navigation" />
    </div>

    <!-- Mobile: top bar and slide-in drawer. The safe-area padding keeps them clear of the notch and
         the home indicator when the app runs from the home screen. -->
    <header class="sticky top-0 z-30 border-b border-slate-200 bg-white pt-[env(safe-area-inset-top)] lg:hidden">
      <div class="flex h-14 items-center gap-2 px-4">
        <AppButton
          variant="ghost"
          size="touch"
          icon
          class="-ml-3"
          aria-label="Open navigation"
          :aria-expanded="drawerOpen"
          @click="drawerOpen = true"
        >
          <Bars3Icon class="size-6" aria-hidden="true" />
        </AppButton>
        <img src="../assets/logo.svg" alt="" class="size-7" />
        <span class="font-semibold text-slate-900">{{ area }}</span>
      </div>
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
      <div
        v-if="drawerOpen"
        class="fixed inset-y-0 left-0 z-50 w-64 max-w-[85vw] bg-slate-900 pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)] lg:hidden"
      >
        <SideNav :navigation="navigation" />
        <AppButton
          variant="inverse"
          size="md"
          icon
          class="absolute top-[calc(0.75rem+env(safe-area-inset-top))] left-full ml-3 rounded-full! bg-white/10"
          aria-label="Close navigation"
          @click="drawerOpen = false"
        >
          <XMarkIcon class="size-5" aria-hidden="true" />
        </AppButton>
      </div>
    </Transition>

    <main class="px-4 pt-6 pb-[calc(1.5rem+env(safe-area-inset-bottom))] sm:px-6 lg:ml-64 lg:px-8 lg:py-8">
      <!-- Keyed by path so moving between e.g. two trainings mounts a fresh page. -->
      <RouterView :key="route.path" />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Bars3Icon, XMarkIcon } from '@heroicons/vue/24/outline'
import AppButton from '../components/button/AppButton.vue'
import SideNav from '../components/navigation/SideNav.vue'
import { navigation } from '../navigation'

const drawerOpen = ref(false)

// Close the drawer after navigating and when Escape is pressed.
const route = useRoute()
const router = useRouter()
watch(() => route.fullPath, () => (drawerOpen.value = false))

// The area shown in the top bar: the navigation entry the page belongs to, e.g. Trainings also
// for a training's form.
const area = computed(
  () => navigation.find((item) => route.path.startsWith(router.resolve(item.to).path))?.name ?? 'HomeGym',
)

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') drawerOpen.value = false
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>
