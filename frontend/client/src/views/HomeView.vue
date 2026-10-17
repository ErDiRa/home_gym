<template>
  <section>
    <p class="text-sm text-slate-500">Welcome back</p>
    <h1 class="truncate text-2xl font-bold tracking-tight">{{ state.user?.email }}</h1>

    <p v-if="loading && !trainings.length" class="mt-6 text-center text-sm text-slate-500">Loading your training…</p>

    <div v-else-if="error" class="mt-6 rounded-2xl border border-red-200 bg-red-50 p-6 text-center">
      <p class="font-medium text-red-800">{{ error }}</p>
      <AppButton variant="secondary" size="md" class="mt-3" @click="load">Try again</AppButton>
    </div>

    <div v-else-if="!trainings.length" class="mt-6 rounded-2xl border border-dashed border-slate-300 bg-white p-6 text-center">
      <p class="font-medium text-slate-800">Your training will appear here</p>
      <p class="mt-1 text-sm text-slate-500">Once your coach assigns one, you'll see today's workout.</p>
    </div>

    <section v-for="plan in plans" :key="plan.training.id" class="mt-6" :aria-labelledby="`training-${plan.training.id}`">
      <h2 :id="`training-${plan.training.id}`" class="text-lg font-semibold text-slate-900">{{ plan.training.name }}</h2>
      <p class="text-sm text-slate-500">{{ formatPeriod(plan.training.start_date, plan.training.end_date) }}</p>
      <p v-if="plan.training.description" class="mt-1 text-sm text-slate-600">{{ plan.training.description }}</p>

      <button
        v-if="plan.past.length"
        type="button"
        class="mt-3 flex min-h-11 w-full items-center justify-center rounded-xl text-sm font-medium text-indigo-600 active:bg-indigo-50"
        :aria-expanded="showPast.has(plan.training.id)"
        @click="togglePast(plan.training.id)"
      >
        {{ showPast.has(plan.training.id) ? 'Hide' : 'Show' }} {{ plan.past.length }} earlier
        {{ plan.past.length === 1 ? 'day' : 'days' }}
      </button>

      <ol class="mt-3 space-y-3">
        <template v-if="showPast.has(plan.training.id)">
          <SessionCard v-for="s in plan.past" :key="s.date" :session="s" class="opacity-60" />
        </template>
        <SessionCard v-for="s in plan.upcoming" :key="s.date" :session="s" :today="s.date === todayDate" />
      </ol>

      <p v-if="!plan.upcoming.length" class="mt-3 rounded-2xl border border-dashed border-slate-300 bg-white p-4 text-center text-sm text-slate-500">
        No more training days in this period.
      </p>
    </section>
  </section>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, onMounted, reactive, ref } from 'vue'
import AppButton from '../components/AppButton.vue'
import SessionCard from '../components/SessionCard.vue'
import { formatPeriod, today } from '../dates'
import { sessions } from '../schedule'
import { state } from '../session'
import { useTrainingsStore } from '../stores/trainings'

const store = useTrainingsStore()
const { trainings, loading } = storeToRefs(store)
const error = ref('')
const todayDate = today()
// Trainings whose earlier days are shown; hidden by default so today is at the top.
const showPast = reactive(new Set<string>())

// Each training's days, split into those before today and the rest.
const plans = computed(() =>
  trainings.value.map((training) => {
    const all = sessions(training)
    return {
      training,
      past: all.filter((s) => s.date < todayDate),
      upcoming: all.filter((s) => s.date >= todayDate),
    }
  }),
)

function togglePast(id: string) {
  if (!showPast.delete(id)) showPast.add(id)
}

async function load() {
  error.value = ''
  try {
    await store.loadTrainings()
  } catch {
    error.value = 'Could not load your training.'
  }
}

onMounted(load)
</script>
