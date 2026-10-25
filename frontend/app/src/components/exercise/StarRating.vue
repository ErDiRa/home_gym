<template>
  <div class="flex" role="radiogroup" :aria-label="label">
    <AppButton
      v-for="star in 5"
      :key="star"
      variant="ghost"
      :size="size === 'touch' ? 'touch' : 'xs'"
      icon
      role="radio"
      :aria-checked="model === star"
      :aria-label="`${star} of 5`"
      :disabled="disabled"
      @click="model = star"
    >
      <StarIcon
        :class="[star <= model ? 'text-amber-400' : 'text-slate-200', size === 'touch' ? 'size-7' : 'size-4']"
        aria-hidden="true"
      />
    </AppButton>
  </div>
</template>

<script setup lang="ts">
import { StarIcon } from '@heroicons/vue/24/solid'
import AppButton from '../button/AppButton.vue'

// A 1-5 star rating to pick (0 = not rated yet): touch for the tracker, sm for lists. label is
// the group's accessible name, e.g. "Rating for Squat".
withDefaults(defineProps<{ label: string; size?: 'touch' | 'sm'; disabled?: boolean }>(), { size: 'touch', disabled: false })
const model = defineModel<number>({ required: true })
</script>
