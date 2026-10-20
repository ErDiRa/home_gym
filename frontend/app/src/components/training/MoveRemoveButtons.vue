<template>
  <div class="flex shrink-0 items-center">
    <AppButton variant="ghost" size="xs" icon :disabled="index === 0" :aria-label="`Move ${label} up`" @click="emit('move', -1)">
      <ChevronUpIcon class="size-4" aria-hidden="true" />
    </AppButton>
    <AppButton
      variant="ghost"
      size="xs"
      icon
      :disabled="index === count - 1"
      :aria-label="`Move ${label} down`"
      @click="emit('move', 1)"
    >
      <ChevronDownIcon class="size-4" aria-hidden="true" />
    </AppButton>
    <AppButton variant="ghost" size="xs" icon :aria-label="`Remove ${label}`" @click="emit('remove')">
      <XMarkIcon class="size-4" aria-hidden="true" />
    </AppButton>
  </div>
</template>

<script setup lang="ts">
import { ChevronDownIcon, ChevronUpIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import AppButton from '../button/AppButton.vue'

// Up / down / remove controls for an item in an ordered list.
defineProps<{ index: number; count: number; label: string }>()
const emit = defineEmits<{ move: [by: -1 | 1]; remove: [] }>()
</script>
