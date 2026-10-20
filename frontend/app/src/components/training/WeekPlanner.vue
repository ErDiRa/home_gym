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
          <li v-for="(slot, s) in week.slots" :key="slot.key">
            <AppBadge
              color="indigo"
              size="md"
              :actions="slotActions(week, w, s)"
              removable
              :remove-label="`Remove ${nameOf(slot.sessionKey)} from week ${w + 1}`"
              @remove="week.slots.splice(s, 1)"
            >
              <span class="max-w-48 truncate">{{ nameOf(slot.sessionKey) }}</span>
            </AppBadge>
          </li>
          <li v-if="!week.slots.length" class="text-sm text-slate-400">Rest week</li>
          <li>
            <ActionMenu
              :items="sessionItems"
              variant="ghost"
              size="xs"
              :disabled="week.slots.length >= MAX_SLOTS"
              :title="week.slots.length >= MAX_SLOTS ? `Max ${MAX_SLOTS} a week` : undefined"
              :aria-label="`Add a session to week ${w + 1}`"
              @select="add(week, $event)"
            >
              <PlusIcon class="size-4" aria-hidden="true" />
              Add session
            </ActionMenu>
          </li>
        </ol>
        <AppButton v-if="w > 0" variant="ghost" size="xs" :title="`Plans week ${w + 1} like week ${w}`" @click="copyWeek(w - 1, w)">
          Copy previous
        </AppButton>
      </div>
      <AppButton v-if="weeks.length > 1" variant="secondary" size="sm" @click="copyFirstToAll">
        <DocumentDuplicateIcon class="size-4" aria-hidden="true" />
        Copy week 1 to all weeks
      </AppButton>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/vue/16/solid'
import { DocumentDuplicateIcon, PlusIcon } from '@heroicons/vue/24/outline'
import { confirmDialog } from '../../confirm'
import { addDays, formatDate } from '../../dates'
import AppButton from '../button/AppButton.vue'
import ActionMenu from '../ui/ActionMenu.vue'
import AppBadge, { type BadgeAction } from '../ui/AppBadge.vue'
import { MAX_SLOTS, move, nextKey, type SessionRow, type WeekRow } from './trainingForm'

// The week plan of a training: for every week the sessions in order, picked from the
// training's sessions (up to 7, repeats allowed). Edited in place; the parent owns the rows.
const props = defineProps<{ sessions: SessionRow[]; startDate: string }>()
const weeks = defineModel<WeekRow[]>({ required: true })

const nameOf = (key: number) => props.sessions.find((s) => s.key === key)?.name || 'Unnamed session'
// E.g. "19 Oct – 25 Oct".
const weekLabel = (w: number) => {
  const monday = addDays(props.startDate, w * 7)
  const short: Intl.DateTimeFormatOptions = { day: 'numeric', month: 'short' }
  return `${formatDate(monday, short)} – ${formatDate(addDays(monday, 6), short)}`
}

// The training's sessions to pick from; their keys are numbers, the menu's ids strings.
const sessionItems = computed(() => props.sessions.map((s) => ({ id: String(s.key), name: s.name || 'Unnamed session' })))

function add(week: WeekRow, id: string) {
  if (week.slots.length < MAX_SLOTS) week.slots.push({ key: nextKey(), sessionKey: Number(id) })
}

// Moves a planned session earlier or later in its week.
function slotActions(week: WeekRow, w: number, s: number): BadgeAction[] {
  const name = nameOf(week.slots[s].sessionKey)
  return [
    {
      icon: ChevronLeftIcon,
      label: `Move ${name} earlier in week ${w + 1}`,
      disabled: s === 0,
      onClick: () => move(week.slots, s, -1),
    },
    {
      icon: ChevronRightIcon,
      label: `Move ${name} later in week ${w + 1}`,
      disabled: s === week.slots.length - 1,
      onClick: () => move(week.slots, s, 1),
    },
  ]
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

async function copyFirstToAll() {
  if (
    weeks.value.slice(1).some((w) => w.slots.length) &&
    !(await confirmDialog({
      title: 'Copy week 1 to all weeks?',
      message: 'This replaces the plans of weeks 2 and later with week 1.',
      confirmLabel: 'Replace',
    }))
  )
    return
  for (let w = 1; w < weeks.value.length; w++) copyWeek(0, w)
}
</script>
