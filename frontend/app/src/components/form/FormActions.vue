<template>
  <!-- On phones and tablets fixed to the bottom of the screen, above the home indicator (fixed, not
       sticky: iOS drags a sticky bar along when the page bounces); from lg it sticks to the bottom of
       the form. The layout keeps the page's end clear of it (.form-actions in style.css). -->
  <div
    class="form-actions fixed inset-x-0 bottom-0 z-20 border-t border-slate-200 bg-slate-50/95 px-4 pt-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] backdrop-blur sm:px-6 lg:sticky lg:inset-x-auto lg:px-0 lg:py-4"
  >
    <div class="flex flex-wrap items-center justify-end gap-x-3 gap-y-2">
      <p v-if="problem" class="w-full text-sm text-amber-700 lg:mr-auto lg:w-auto">{{ problem }}</p>
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
// A form's Save / Cancel bar, always in reach; problem says why saving is disabled, if it is.
defineProps<{ problem?: string }>()
</script>
