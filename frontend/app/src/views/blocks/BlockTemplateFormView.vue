<template>
  <div class="max-w-3xl">
    <RouterLink
      :to="{ name: 'blocks' }"
      class="mb-4 inline-flex items-center gap-x-1.5 text-sm font-medium text-slate-600 hover:text-indigo-600"
    >
      <ArrowLeftIcon class="size-4" aria-hidden="true" />
      Back to saved blocks
    </RouterLink>

    <div class="mb-1 flex flex-wrap items-center justify-between gap-4">
      <h1 class="text-xl font-bold text-slate-900">{{ templateId ? 'Edit saved block' : 'New saved block' }}</h1>
      <AppButton v-if="templateId" variant="secondary" size="md" :disabled="saving" @click="remove">
        <TrashIcon class="size-4" aria-hidden="true" />
        Delete
      </AppButton>
    </div>
    <p class="mt-1 mb-6 text-sm text-slate-500">
      A combination of exercises to add to sessions. Sessions it was added to keep their own block, so changes here only reach
      sessions you add it to from now on.
    </p>

    <form v-if="loaded" class="space-y-4" @submit.prevent="submit">
      <div class="space-y-4 rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
        <label class="block text-sm font-medium text-slate-700">
          Name<span class="text-red-600" aria-hidden="true"> *</span>
          <TextInput v-model="name" type="text" maxlength="100" required placeholder="e.g. Clean complex" class="mt-1" />
        </label>
        <fieldset>
          <legend class="mb-1 text-sm font-medium text-slate-700">Kind</legend>
          <SegmentedControl v-model="kind" :options="kindOptions" label="Kind" />
        </fieldset>

        <div>
          <h2 class="mb-2 text-sm font-medium text-slate-700">Exercises</h2>
          <ol v-if="exercises.length" class="mb-3 space-y-2">
            <li
              v-for="(e, i) in exercises"
              :key="e.key"
              class="flex items-center gap-2 rounded-md border border-slate-200 bg-slate-50 px-2 py-1.5"
            >
              <span class="w-6 shrink-0 text-sm font-semibold text-indigo-600">{{ i + 1 }}</span>
              <span class="min-w-0 flex-1 truncate text-sm font-medium text-slate-900">{{ e.name }}</span>
              <ExerciseTypeBadge :type="e.type" />
              <MoveRemoveButtons
                :index="i"
                :count="exercises.length"
                :label="e.name"
                @move="move(exercises, i, $event)"
                @remove="exercises.splice(i, 1)"
              />
            </li>
          </ol>
          <AppButton v-if="remaining !== 0" variant="secondary" size="sm" @click="pickerOpen = true">
            <PlusIcon class="size-4" aria-hidden="true" />
            Add exercises…
          </AppButton>
        </div>
      </div>

      <FormActions :problem="problem">
        <AppButton variant="secondary" :disabled="saving" @click="goBack">Cancel</AppButton>
        <AppButton type="submit" :disabled="saving || !!problem">
          {{ saving ? 'Saving…' : templateId ? 'Save' : 'Create' }}
        </AppButton>
      </FormActions>
    </form>

    <ExercisePickerDialog
      :open="pickerOpen"
      title="Add exercises to the block"
      :in-block="exercises.map((e) => e.exercise_id)"
      :limit="remaining"
      @close="pickerOpen = false"
      @add="addExercises"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { ArrowLeftIcon, PlusIcon, TrashIcon } from '@heroicons/vue/24/outline'
import { ApiError } from '../../api'
import AppButton from '../../components/button/AppButton.vue'
import ExerciseTypeBadge from '../../components/exercise/ExerciseTypeBadge.vue'
import FormActions from '../../components/form/FormActions.vue'
import SegmentedControl from '../../components/form/SegmentedControl.vue'
import TextInput from '../../components/form/TextInput.vue'
import ExercisePickerDialog from '../../components/training/ExercisePickerDialog.vue'
import { MAX_TEMPLATE_EXERCISES, move, nextKey, templateProblem } from '../../components/training/trainingForm'
import MoveRemoveButtons from '../../components/ui/MoveRemoveButtons.vue'
import { confirmDialog } from '../../confirm'
import { useBlockTemplatesStore } from '../../stores/blockTemplates'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import type { Exercise, ExerciseType } from '../../types'

const router = useRouter()
const route = useRoute()
const store = useBlockTemplatesStore()
const exercisesStore = useExercisesStore()
const lookups = useLookupsStore()
// Set when editing a saved block (route /blocks/:id), absent when creating one.
const templateId = route.params.id as string | undefined

interface Row {
  key: number
  exercise_id: string
  name: string
  type: ExerciseType
}

const name = ref('')
const kind = ref('')
const exercises = ref<Row[]>([])
const loaded = ref(false)
const saving = ref(false)
const pickerOpen = ref(false)

const kindOptions = computed(() => lookups.blockKinds.map((k) => ({ value: k.code, label: k.name, title: k.description })))
const problem = computed(() => templateProblem(name.value, kind.value, exercises.value.length))
// How many more exercises fit: the kind's limit, at most the saved blocks' own.
const remaining = computed(() => {
  const max = Math.min(lookups.kindOf(kind.value)?.max_exercises ?? MAX_TEMPLATE_EXERCISES, MAX_TEMPLATE_EXERCISES)
  return Math.max(max - exercises.value.length, 0)
})

onMounted(async () => {
  try {
    const [template] = await Promise.all([
      templateId ? store.getTemplate(templateId) : undefined,
      exercisesStore.loadExercises(),
      exercisesStore.loadRatings(),
      lookups.loadTrainingLookups(),
    ])
    if (template) {
      name.value = template.name
      kind.value = template.kind
      exercises.value = template.exercises.map((e) => ({ key: nextKey(), ...e }))
    } else {
      // A complex is the most common combination to keep.
      kind.value = lookups.kindOf('complex') ? 'complex' : (lookups.blockKinds[0]?.code ?? '')
    }
    loaded.value = true
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 404 ? 'Saved block not found.' : 'Could not load the saved block.')
  }
})

function addExercises(picked: Exercise[]) {
  exercises.value.push(...picked.map((e) => ({ key: nextKey(), exercise_id: e.id, name: e.name, type: e.type })))
  pickerOpen.value = false
}

async function submit() {
  if (problem.value) return
  const input = { name: name.value, kind: kind.value, exercise_ids: exercises.value.map((e) => e.exercise_id) }
  saving.value = true
  try {
    if (templateId) {
      // Stay on the page so further changes can follow.
      await store.updateTemplate(templateId, input)
      toast.success('Saved block updated.')
    } else {
      await store.createTemplate(input)
      toast.success('Saved block created.')
      await goBack()
    }
  } catch (e) {
    toast.error(e instanceof ApiError && (e.status === 400 || e.status === 409) ? e.message : 'Could not save the block.')
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!templateId) return
  const ok = await confirmDialog({
    title: `Delete the saved block "${name.value}"?`,
    message: 'Sessions it was added to keep their blocks.',
    confirmLabel: 'Delete',
    danger: true,
  })
  if (!ok) return
  saving.value = true
  try {
    await store.deleteTemplate(templateId)
    toast.success('Saved block deleted.')
    await goBack()
  } catch {
    toast.error('Could not delete the saved block.')
  } finally {
    saving.value = false
  }
}

function goBack() {
  return router.push({ name: 'blocks' })
}
</script>
