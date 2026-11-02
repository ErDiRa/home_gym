<template>
  <section>
    <h1 class="text-2xl font-bold tracking-tight text-slate-900">Nutrition</h1>

    <!-- The week shown: the current one by default, its Monday in the URL (?week=). -->
    <nav class="mt-4 flex items-center gap-1" aria-label="Weeks">
      <AppButton variant="ghost" size="touch" icon aria-label="Previous week" @click="shift(-1)">
        <ChevronLeftIcon class="size-5" aria-hidden="true" />
      </AppButton>
      <div class="min-w-0 flex-1 px-1">
        <h2 id="nutrition-week" class="font-semibold text-slate-900">{{ weekDates }}</h2>
        <p v-if="averages" class="text-xs text-slate-500">
          Average of {{ averages.count }} {{ averages.count === 1 ? 'day' : 'days' }}: {{ formatAmount(averages.kcal) }} kcal ·
          P {{ formatAmount(averages.protein) }} g
        </p>
      </div>
      <AppButton v-if="!isCurrent" variant="text" size="touch" @click="shift(0)">Current</AppButton>
      <AppButton variant="ghost" size="touch" icon aria-label="Next week" @click="shift(1)">
        <ChevronRightIcon class="size-5" aria-hidden="true" />
      </AppButton>
    </nav>

    <!-- Without a body and a weight there are no targets. -->
    <RouterLink
      v-if="days.length && !days.some((d) => d.target)"
      :to="{ name: 'track-settings' }"
      class="mt-3 flex min-h-14 items-center gap-3 rounded-xl border border-indigo-200 bg-indigo-50 px-4 py-3 text-sm text-indigo-900 active:bg-indigo-100"
    >
      <span class="flex-1">Enter your body and weight in the settings to get daily targets.</span>
      <ChevronRightIcon class="size-5 shrink-0 text-indigo-400" aria-hidden="true" />
    </RouterLink>

    <p v-if="loading && !days.length" class="mt-6 text-center text-sm text-slate-500">Loading your week…</p>
    <div v-else-if="error" class="mt-6 rounded-2xl border border-red-200 bg-red-50 p-6 text-center">
      <p class="font-medium text-red-800">{{ error }}</p>
      <AppButton variant="secondary" size="touch" class="mt-3" @click="load">Try again</AppButton>
    </div>

    <!-- One card per day: tapping it logs or corrects its macros. Future days only show. -->
    <ul v-else class="mt-3 space-y-2" aria-labelledby="nutrition-week">
      <li v-for="d in days" :key="d.date">
        <button
          type="button"
          :disabled="d.date > today()"
          :class="[
            'block w-full rounded-xl border bg-white px-4 py-3 text-left active:bg-slate-50 disabled:opacity-50',
            d.date === today() ? 'border-indigo-300 ring-1 ring-indigo-200' : 'border-slate-200',
          ]"
          @click="edit(d)"
        >
          <span class="flex items-center gap-2">
            <span class="flex-1 font-medium text-slate-900">
              {{ d.date === today() ? 'Today' : formatDate(d.date, { weekday: true, year: false }) }}
            </span>
            <span v-if="d.workouts.length" class="flex min-w-0 items-center gap-1 text-xs text-slate-600">
              <BoltIcon class="size-4 shrink-0 text-indigo-500" aria-hidden="true" />
              <span class="truncate">{{ d.workouts.map((w) => w.session_name).join(', ') }}</span>
            </span>
            <span v-else class="text-xs text-slate-400">Rest day</span>
          </span>
          <MacroBars v-if="d.macros || d.target" :macros="d.macros" :target="d.target" class="mt-2" />
          <span v-else class="mt-1 block text-sm text-slate-500">Nothing logged</span>
        </button>
      </li>
    </ul>

    <MacroDaySheet
      :open="sheetOpen"
      :date="editing?.date ?? today()"
      :macros="editing?.macros ?? null"
      :target="editing?.target ?? null"
      @close="sheetOpen = false"
      @saved="saved"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { BoltIcon, ChevronLeftIcon, ChevronRightIcon } from '@heroicons/vue/24/outline'
import AppButton from '../../components/button/AppButton.vue'
import MacroBars from '../../components/nutrition/MacroBars.vue'
import MacroDaySheet from '../../components/nutrition/MacroDaySheet.vue'
import { formatAmount } from '../../components/nutrition/macros'
import { addDays, formatDate, isMonday, toIso, today, weekStart } from '../../dates'
import { useNutritionStore } from '../../stores/nutrition'
import type { DayMacros, NutritionDay } from '../../types'

const route = useRoute()
const router = useRouter()
const store = useNutritionStore()

const days = ref<NutritionDay[]>([])
const loading = ref(false)
const error = ref('')
// The day in the sheet; kept while it closes.
const editing = ref<NutritionDay | null>(null)
const sheetOpen = ref(false)

const thisMonday = toIso(weekStart())
const monday = computed(() => {
  const w = route.query.week
  return typeof w === 'string' && isMonday(w) ? w : thisMonday
})
const isCurrent = computed(() => monday.value === thisMonday)
const weekDates = computed(() => `${formatDate(monday.value, { year: false })} – ${formatDate(addDays(monday.value, 6))}`)

// The average of the days logged in the week.
const averages = computed(() => {
  const logged = days.value.filter((d) => d.macros).map((d) => d.macros as DayMacros)
  if (!logged.length) return null
  const sum = (f: (m: DayMacros) => number) => logged.reduce((s, m) => s + f(m), 0) / logged.length
  return { count: logged.length, kcal: sum((m) => m.kcal), protein: sum((m) => m.protein_g ?? 0) }
})

let request = 0

// Only the latest request is kept, so stepping quickly through the weeks can't show an older answer.
async function load() {
  const id = ++request
  error.value = ''
  loading.value = true
  try {
    const result = await store.listNutritionDays(monday.value, addDays(monday.value, 6))
    if (id === request) days.value = result
  } catch {
    if (id === request) error.value = 'Could not load your week.'
  } finally {
    if (id === request) loading.value = false
  }
}

function edit(d: NutritionDay) {
  editing.value = d
  sheetOpen.value = true
}

// The sheet saved (or cleared) a day; the targets don't change with the macros.
function saved(macros: DayMacros | null) {
  const date = editing.value?.date
  days.value = days.value.map((d) => (d.date === date ? { ...d, macros } : d))
}

// Steps the given number of weeks from the one shown; 0 goes back to the current week.
function shift(weeks: number) {
  const target = weeks === 0 ? thisMonday : addDays(monday.value, weeks * 7)
  router.replace({ query: { ...route.query, week: target === thisMonday ? undefined : target } })
}

watch(monday, load, { immediate: true })
</script>
