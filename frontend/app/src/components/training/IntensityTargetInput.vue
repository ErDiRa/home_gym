<template>
  <fieldset class="min-w-0">
    <legend class="mb-1 text-xs font-medium text-slate-600">
      Target <span class="font-normal text-slate-400">· work and recovery</span>
    </legend>
    <div class="flex flex-wrap items-center gap-1">
      <div class="w-24">
        <SelectField
          :model-value="target.intensity_metric"
          :options="intensityMetrics"
          :searchable="false"
          placeholder="Metric"
          size="sm"
          :aria-label="`${label} target metric`"
          @update:model-value="target.intensity_metric = $event as IntensityMetric | null"
        />
      </div>
      <IntensityInput v-model="target.intensity_value" :metric="target.intensity_metric" :aria-label="`${label} target`" class="w-16!" />
      <span class="text-sm text-slate-400" aria-hidden="true">–</span>
      <IntensityInput
        v-model="target.intensity_max"
        :metric="target.intensity_metric"
        placeholder="max"
        :aria-label="`${label} target, upper end of a range (optional)`"
        class="w-16!"
      />
      <span class="ml-2 text-xs font-medium text-slate-600">Recovery</span>
      <IntensityInput v-model="target.recovery_value" :metric="target.intensity_metric" :aria-label="`${label} recovery`" class="w-16!" />
      <span class="text-sm text-slate-400" aria-hidden="true">–</span>
      <IntensityInput
        v-model="target.recovery_max"
        :metric="target.intensity_metric"
        placeholder="max"
        :aria-label="`${label} recovery, upper end of a range (optional)`"
        class="w-16!"
      />
    </div>
  </fieldset>
</template>

<script setup lang="ts">
import IntensityInput from '../form/IntensityInput.vue'
import SelectField from '../form/SelectField.vue'
import { intensityMetrics } from '../exercise/intensity'
import type { Intensity, IntensityMetric } from '../../types'

// Edits the target intensity of an endurance exercise in place: one metric (watts, cal/h or a
// pace), the work's average and the recovery's between intervals, each optionally a range.
defineProps<{ label: string }>()
const target = defineModel<Intensity>({ required: true })
</script>
