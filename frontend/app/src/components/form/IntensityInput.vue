<template>
  <TextInput
    :model-value="text"
    type="text"
    inputmode="decimal"
    autocomplete="off"
    :placeholder="placeholder ?? (isPace(metric) ? '1:55' : '—')"
    :size="size"
    :tone="tone"
    @update:model-value="typed"
    @focus="focused = true"
    @blur="blurred"
  />
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import TextInput, { type InputSize } from './TextInput.vue'
import { formatIntensityValue, isPace, parseIntensity } from '../exercise/intensity'
import type { IntensityMetric } from '../../types'

// An intensity value in its metric: watts and cal/h as numbers, a pace typed as m:ss (or 1.55,
// as phones' number pads have no colon) and bound to seconds; null while empty or unreadable. The
// text is formatted again once the input is left.
const props = withDefaults(
  defineProps<{ metric: IntensityMetric | null; size?: InputSize; tone?: 'default' | 'success'; placeholder?: string }>(),
  { size: 'sm', tone: 'default', placeholder: undefined },
)
const model = defineModel<number | null>({ required: true })

const format = (v: number | null) => (v === null ? '' : formatIntensityValue(v, props.metric))
const text = ref(format(model.value))
const focused = ref(false)

function typed(value: string | number | null | undefined) {
  text.value = String(value ?? '')
  model.value = parseIntensity(text.value, props.metric)
}

function blurred() {
  focused.value = false
  text.value = format(model.value)
}

// A value set from outside (prefilled, converted, undone) shows unless it's being typed.
watch([model, () => props.metric], () => {
  if (!focused.value) text.value = format(model.value)
})
</script>
