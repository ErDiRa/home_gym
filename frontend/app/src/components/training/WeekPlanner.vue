<template>
  <div class="space-y-2">
    <p v-if="!sessions.length" class="rounded-lg border border-dashed border-slate-300 px-4 py-6 text-center text-sm text-slate-500">
      Add sessions above to plan the weeks.
    </p>
    <template v-else>
      <div
        v-for="(week, w) in weeks"
        :key="week.key"
        class="flex flex-wrap items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2"
      >
        <div class="w-24 shrink-0">
          <p class="text-sm font-semibold text-slate-800">Week {{ w + 1 }}</p>
          <p v-if="startDate" class="text-xs text-slate-500">{{ weekLabel(w) }}</p>
        </div>
        <ol class="flex min-w-0 flex-1 flex-wrap items-center gap-1.5" :aria-label="`Sessions of week ${w + 1}`">
          <li
            v-for="(slot, s) in week.slots"
            :key="slot.key"
            class="flex items-center gap-0.5 rounded-full border border-indigo-200 bg-indigo-50 py-0.5 pr-1 pl-2.5 text-sm text-indigo-900"
          >
            <span class="max-w-48 truncate">{{ nameOf(slot.sessionKey) }}</span>
            <button
              type="button"
              :disabled="s === 0"
              :class="chipButton"
              :aria-label="`Move ${nameOf(slot.sessionKey)} earlier in week ${w + 1}`"
              @click="move(week.slots, s, -1)"
            >
              <ChevronLeftIcon class="size-3.5" aria-hidden="true" />
            </button>
            <button
              type="button"
              :disabled="s === week.slots.length - 1"
              :class="chipButton"
              :aria-label="`Move ${nameOf(slot.sessionKey)} later in week ${w + 1}`"
              @click="move(week.slots, s, 1)"
            >
              <ChevronRightIcon class="size-3.5" aria-hidden="true" />
            </button>
            <button
              type="button"
              :class="chipButton"
              :aria-label="`Remove ${nameOf(slot.sessionKey)} from week ${w + 1}`"
              @click="week.slots.splice(s, 1)"
            >
              <XMarkIcon class="size-3.5" aria-hidden="true" />
            </button>
          </li>
          <li v-if="!week.slots.length" class="text-sm text-slate-400">Rest week</li>
          <li>
            <select
              :disabled="week.slots.length >= MAX_SLOTS"
              :aria-label="`Add a session to week ${w + 1}`"
              class="rounded-full border border-dashed border-slate-300 bg-white py-1 pr-7 pl-2.5 text-sm text-slate-600 hover:border-indigo-400 disabled:opacity-50"
              @change="add(week, $event)"
            >
              <option value="">{{ week.slots.length >= MAX_SLOTS ? `Max ${MAX_SLOTS} a week` : '+ Add session' }}</option>
              <option v-for="session in sessions" :key="session.key" :value="session.key">
                {{ session.name || 'Unnamed session' }}
              </option>
            </select>
          </li>
        </ol>
        <button
          v-if="w > 0"
          type="button"
          class="shrink-0 rounded px-2 py-1 text-xs font-medium text-slate-500 hover:bg-slate-100 hover:text-slate-800"
          :title="`Plans week ${w + 1} like week ${w}`"
          @click="copyWeek(w - 1, w)"
        >
          Copy previous
        </button>
      </div>
      <AppButton v-if="weeks.length > 1" variant="secondary" size="sm" @click="copyFirstToAll">
        <DocumentDuplicateIcon class="size-4" aria-hidden="true" />
        Copy week 1 to all weeks
      </AppButton>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ChevronLeftIcon, ChevronRightIcon, DocumentDuplicateIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import { addDays, formatDate } from '../../dates'
import AppButton from '../button/AppButton.vue'
import { MAX_SLOTS, move, nextKey, type SessionRow, type WeekRow } from './trainingForm'

// The week plan of a training: for every week the sessions in order, picked from the
// training's sessions (up to 7, repeats allowed). Edited in place; the parent owns the rows.
const props = defineProps<{ sessions: SessionRow[]; startDate: string }>()
const weeks = defineModel<WeekRow[]>({ required: true })

const chipButton = 'rounded-full p-0.5 text-indigo-500 hover:bg-indigo-100 hover:text-indigo-900 disabled:opacity-30'

const nameOf = (key: number) => props.sessions.find((s) => s.key === key)?.name || 'Unnamed session'
// E.g. "19 Oct – 25 Oct".
const weekLabel = (w: number) => {
  const monday = addDays(props.startDate, w * 7)
  const short: Intl.DateTimeFormatOptions = { day: 'numeric', month: 'short' }
  return `${formatDate(monday, short)} – ${formatDate(addDays(monday, 6), short)}`
}

function add(week: WeekRow, event: Event) {
  const select = event.target as HTMLSelectElement
  const key = Number(select.value)
  select.value = '' // back to the prompt, ready for the next one
  if (key && week.slots.length < MAX_SLOTS) week.slots.push({ key: nextKey(), sessionKey: key })
}

// Copies the sessions of one week into another. Where the target already plans the same
// session at that place, its saved slot is kept, so the workouts done of it stay linked.
function copyWeek(from: number, to: number) {
  const target = weeks.value[to].slots
  weeks.value[to].slots = weeks.value[from].slots.map((s, i) => ({
    key: nextKey(),
    id: target[i]?.sessionKey === s.sessionKey ? target[i].id : undefined,
    sessionKey: s.sessionKey,
  }))
}

function copyFirstToAll() {
  if (weeks.value.slice(1).some((w) => w.slots.length) && !confirm('Replace the plans of weeks 2 and later with week 1?')) return
  for (let w = 1; w < weeks.value.length; w++) copyWeek(0, w)
}
</script>
