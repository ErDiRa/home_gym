<template>
  <Dialog :open="open" :initial-focus="searchInput?.el" class="relative z-50" @close="emit('close')">
    <div class="fixed inset-0 bg-slate-900/40" aria-hidden="true" />

    <div class="fixed inset-0 flex items-start justify-center p-4 sm:pt-[8vh]">
      <DialogPanel class="flex max-h-[84vh] w-full max-w-3xl flex-col rounded-xl bg-white shadow-2xl">
        <div class="border-b border-slate-200 p-4">
          <div class="mb-3 flex items-center justify-between gap-4">
            <DialogTitle class="text-base font-semibold text-slate-900">{{ title }}</DialogTitle>
            <AppButton variant="ghost" size="sm" icon aria-label="Close" @click="emit('close')">
              <XMarkIcon class="size-5" aria-hidden="true" />
            </AppButton>
          </div>

          <div class="relative">
            <MagnifyingGlassIcon class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-slate-400" aria-hidden="true" />
            <TextInput
              ref="searchInput"
              v-model="query"
              type="search"
              placeholder="Search by name, muscle group, joint or focus…"
              role="combobox"
              aria-expanded="true"
              aria-controls="exercise-picker-results"
              :aria-activedescendant="filtered[active] ? optionId(filtered[active].id) : undefined"
              class="pl-9!"
              @keydown="onKeydown"
            />
          </div>

          <div class="mt-3 grid gap-3 sm:grid-cols-3">
            <MultiSelect v-model="muscleFilter" label="Muscle groups" :options="muscleOptions" placeholder="Any" />
            <MultiSelect v-model="jointFilter" label="Joints" :options="jointOptions" placeholder="Any" />
            <MultiSelect v-model="focusFilter" label="Focus" :options="focusOptions" placeholder="Any" />
          </div>

          <div class="mt-3 flex flex-wrap items-center gap-2 text-sm">
            <AppButton
              v-for="(label, type) in exerciseTypeLabels"
              :key="type"
              variant="secondary"
              size="xs"
              :pressed="typeFilter.includes(type)"
              @click="toggleType(type)"
            >
              {{ label }}
            </AppButton>
            <template v-if="ratings.size">
              <span class="ml-2 text-xs font-medium text-slate-600">Your rating</span>
              <AppButton
                v-for="option in ratingOptions"
                :key="option.value"
                variant="secondary"
                size="xs"
                class="gap-0.5"
                :pressed="ratingFilter.includes(option.value)"
                @click="toggleRating(option.value)"
              >
                <StarIcon v-if="option.value" class="size-3" aria-hidden="true" />
                {{ option.label }}
              </AppButton>
            </template>
            <span class="ml-auto text-slate-500">{{ filtered.length }} {{ filtered.length === 1 ? 'exercise' : 'exercises' }}</span>
            <AppButton v-if="filtering" variant="text" size="xs" @click="clearFilters">Clear filters</AppButton>
          </div>
        </div>

        <ul
          id="exercise-picker-results"
          role="listbox"
          aria-multiselectable="true"
          :aria-label="title"
          class="min-h-40 flex-1 overflow-y-auto py-1"
        >
          <li v-if="!filtered.length" class="px-4 py-6 text-center text-sm text-slate-500">No exercises match.</li>
          <li
            v-for="(exercise, i) in filtered"
            :id="optionId(exercise.id)"
            :key="exercise.id"
            role="option"
            :aria-selected="isSelected(exercise.id)"
            :aria-disabled="!canSelect(exercise.id)"
            :class="[
              'flex cursor-pointer items-start gap-3 px-4 py-2',
              i === active ? 'bg-indigo-50' : '',
              canSelect(exercise.id) ? '' : 'cursor-not-allowed opacity-50',
            ]"
            @mousemove="active = i"
            @click="toggle(exercise.id)"
            @dblclick="emit('add', [exercise])"
          >
            <span
              :class="isSelected(exercise.id) ? 'border-indigo-600 bg-indigo-600 text-white' : 'border-slate-300 bg-white'"
              class="mt-0.5 flex size-4 shrink-0 items-center justify-center rounded border"
              aria-hidden="true"
            >
              <CheckIcon v-if="isSelected(exercise.id)" class="size-3" />
            </span>
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-1.5">
                <span class="font-medium text-slate-900">{{ exercise.name }}</span>
                <ExerciseTypeBadge :type="exercise.type" />
                <RatingBadge v-if="ratings.get(exercise.id)" :rating="ratings.get(exercise.id)!" />
                <AppBadge v-for="focus in exercise.focuses" :key="focus.id" color="indigo" :title="focus.name">
                  {{ focus.alias }}
                </AppBadge>
                <span v-if="inBlockCount(exercise.id)" class="text-xs text-slate-400">
                  in block{{ inBlockCount(exercise.id) > 1 ? ` ×${inBlockCount(exercise.id)}` : '' }}
                </span>
              </div>
              <p v-if="details(exercise)" class="truncate text-xs text-slate-500">{{ details(exercise) }}</p>
            </div>
          </li>
        </ul>

        <div class="border-t border-slate-200 p-4">
          <div v-if="selectedExercises.length" class="mb-3 flex flex-wrap gap-1">
            <AppBadge
              v-for="exercise in selectedExercises"
              :key="exercise.id"
              color="indigo"
              removable
              :remove-label="`Remove ${exercise.name}`"
              @remove="toggle(exercise.id)"
            >
              {{ exercise.name }}
            </AppBadge>
          </div>
          <div class="flex flex-wrap items-center justify-end gap-3">
            <p class="mr-auto text-xs text-slate-500">
              <template v-if="limit != null">This block takes {{ limit }} more {{ limit === 1 ? 'exercise' : 'exercises' }}. </template>
              <kbd class="font-sans">↑↓</kbd> move · <kbd class="font-sans">Enter</kbd> {{ limit === 1 ? 'add' : 'select' }}<template v-if="limit !== 1">
                · <kbd class="font-sans">{{ modKey }}+Enter</kbd> add</template>
            </p>
            <AppButton variant="secondary" size="md" @click="emit('close')">Cancel</AppButton>
            <AppButton size="md" :disabled="!selected.length" @click="addSelected">
              Add {{ selected.length || '' }} {{ selected.length === 1 ? 'exercise' : 'exercises' }}
            </AppButton>
          </div>
        </div>
      </DialogPanel>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, nextTick, ref, watch } from 'vue'
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/vue'
import { CheckIcon, MagnifyingGlassIcon, StarIcon, XMarkIcon } from '@heroicons/vue/16/solid'
import AppButton from '../button/AppButton.vue'
import ExerciseTypeBadge from '../exercise/ExerciseTypeBadge.vue'
import RatingBadge from '../exercise/RatingBadge.vue'
import MultiSelect from '../form/MultiSelect.vue'
import TextInput, { type TextInputExposed } from '../form/TextInput.vue'
import AppBadge from '../ui/AppBadge.vue'
import { useExercisesStore } from '../../stores/exercises'
import { exerciseTypeLabels, type Exercise, type ExerciseType, type Option } from '../../types'

// Finds exercises by search and filters and adds one or several to a block at once. The
// selection survives changing the search, so it can be built up over several searches.
const props = defineProps<{
  open: boolean
  title: string
  inBlock: string[] // exercise ids already in the block
  limit: number | null // how many more the block takes; null = no limit
}>()
const emit = defineEmits<{ close: []; add: [exercises: Exercise[]] }>()

// The user's ratings by exercise id.
const { exercises, ratings } = storeToRefs(useExercisesStore())

const searchInput = ref<TextInputExposed | null>(null)

const query = ref('')
const typeFilter = ref<ExerciseType[]>([])
const muscleFilter = ref<string[]>([]) // names, as exercises list them
const jointFilter = ref<string[]>([]) // names
const focusFilter = ref<string[]>([]) // ids
const ratingFilter = ref<number[]>([]) // 1-5, 0 = not rated
const selected = ref<string[]>([]) // ids, in the order they were picked
const active = ref(0) // index into filtered

// Start fresh every time the dialog opens.
watch(
  () => props.open,
  (open) => {
    if (!open) return
    query.value = ''
    clearFilters()
    selected.value = []
    active.value = 0
  },
)

// Offer only the values exercises actually use.
const byName = (a: Option, b: Option) => a.name.localeCompare(b.name)
const namesOf = (pick: (e: Exercise) => string[]) =>
  [...new Set(exercises.value.flatMap(pick))].map((name) => ({ id: name, name })).sort(byName)
const muscleOptions = computed(() => namesOf((e) => e.muscle_groups ?? []))
const jointOptions = computed(() => namesOf((e) => e.joints ?? []))
const focusOptions = computed(() => {
  const focuses = new Map<string, Option>()
  for (const e of exercises.value)
    for (const f of e.focuses ?? []) focuses.set(f.id, { id: f.id, name: `${f.name} (${f.alias})` })
  return [...focuses.values()].sort(byName)
})

// Rating filter chips: 1-5, plus "Not rated" as 0 so unrated exercises can be filtered too.
const ratingOptions = [1, 2, 3, 4, 5].map((r) => ({ value: r, label: String(r) })).concat({ value: 0, label: 'Not rated' })
const ratingOf = (e: Exercise) => ratings.value.get(e.id)?.rating ?? 0

// What the search matches against, lower-cased once per exercise.
const searchText = computed(
  () =>
    new Map(
      exercises.value.map((e) => [
        e.id,
        [e.name, ...(e.muscle_groups ?? []), ...(e.joints ?? []), ...(e.focuses ?? []).flatMap((f) => [f.name, f.alias])]
          .join(' ')
          .toLowerCase(),
      ]),
    ),
)

// Every search term has to match; within a filter any picked value does, across filters all.
const filtered = computed(() => {
  const terms = query.value.toLowerCase().split(/\s+/).filter(Boolean)
  const anyOf = (picked: string[], values: string[]) => !picked.length || values.some((v) => picked.includes(v))
  return exercises.value.filter(
    (e) =>
      terms.every((t) => searchText.value.get(e.id)!.includes(t)) &&
      (!typeFilter.value.length || typeFilter.value.includes(e.type)) &&
      anyOf(muscleFilter.value, e.muscle_groups ?? []) &&
      anyOf(jointFilter.value, e.joints ?? []) &&
      anyOf(
        focusFilter.value,
        (e.focuses ?? []).map((f) => f.id),
      ) &&
      (!ratingFilter.value.length || ratingFilter.value.includes(ratingOf(e))),
  )
})
watch(filtered, () => (active.value = 0))

const filtering = computed(
  () =>
    typeFilter.value.length +
      muscleFilter.value.length +
      jointFilter.value.length +
      focusFilter.value.length +
      ratingFilter.value.length >
    0,
)

function clearFilters() {
  typeFilter.value = []
  muscleFilter.value = []
  jointFilter.value = []
  focusFilter.value = []
  ratingFilter.value = []
}

function toggleType(type: ExerciseType) {
  typeFilter.value = typeFilter.value.includes(type)
    ? typeFilter.value.filter((t) => t !== type)
    : [...typeFilter.value, type]
}

function toggleRating(rating: number) {
  ratingFilter.value = ratingFilter.value.includes(rating)
    ? ratingFilter.value.filter((r) => r !== rating)
    : [...ratingFilter.value, rating]
}

const exerciseById = computed(() => new Map(exercises.value.map((e) => [e.id, e])))
const selectedExercises = computed(() => selected.value.map((id) => exerciseById.value.get(id)!).filter(Boolean))

const isSelected = (id: string) => selected.value.includes(id)
// A block that takes one more exercise swaps the pick instead of refusing a second one.
const canSelect = (id: string) =>
  props.limit == null || props.limit === 1 || isSelected(id) || selected.value.length < props.limit

function toggle(id: string) {
  if (isSelected(id)) selected.value = selected.value.filter((s) => s !== id)
  else if (props.limit === 1) selected.value = [id]
  else if (canSelect(id)) selected.value = [...selected.value, id]
}

function addSelected() {
  if (selectedExercises.value.length) emit('add', selectedExercises.value)
}

const inBlockCount = (id: string) => props.inBlock.filter((e) => e === id).length
const details = (e: Exercise) => [...(e.muscle_groups ?? []), ...(e.joints ?? [])].join(', ')
const optionId = (id: string) => `exercise-option-${id}`

const modKey = /Mac|iPhone|iPad/.test(navigator.platform) ? '⌘' : 'Ctrl'

// Arrows move through the results, Enter toggles the highlighted one, Cmd/Ctrl+Enter adds the
// selection (or the highlighted exercise when nothing is selected). A block that takes only one
// more exercise adds on Enter right away.
function onKeydown(event: KeyboardEvent) {
  const current = filtered.value[active.value]
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    if (!filtered.value.length) return
    const by = event.key === 'ArrowDown' ? 1 : -1
    active.value = (active.value + by + filtered.value.length) % filtered.value.length
    nextTick(() => document.getElementById(optionId(filtered.value[active.value].id))?.scrollIntoView({ block: 'nearest' }))
  } else if (event.key === 'Enter') {
    event.preventDefault()
    if (event.metaKey || event.ctrlKey) {
      if (selected.value.length) addSelected()
      else if (current) emit('add', [current])
    } else if (current) {
      if (props.limit === 1) emit('add', [current])
      else toggle(current.id)
    }
  }
}
</script>
