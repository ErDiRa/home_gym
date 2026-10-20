<template>
  <div class="max-w-4xl">
    <RouterLink
      :to="{ name: 'sessions' }"
      class="mb-4 inline-flex items-center gap-x-1.5 text-sm font-medium text-slate-600 hover:text-indigo-600"
    >
      <ArrowLeftIcon class="size-4" aria-hidden="true" />
      Back to sessions
    </RouterLink>

    <div class="mb-1 flex flex-wrap items-center justify-between gap-4">
      <h1 class="text-xl font-bold text-slate-900">{{ sessionId ? 'Edit session' : 'New session' }}</h1>
      <AppButton v-if="sessionId" variant="secondary" size="md" :disabled="saving" @click="remove">
        <TrashIcon class="size-4" aria-hidden="true" />
        Delete
      </AppButton>
    </div>
    <p class="mt-1 mb-6 text-sm text-slate-500">
      A template for your trainings. Trainings it was added to keep their own copy, so changes here only reach trainings you add it
      to from now on.
    </p>

    <form class="space-y-4" @submit.prevent="submit">
      <SessionEditor v-if="session" v-model="session" />

      <!-- Sticks to the bottom of the screen so saving doesn't need scrolling down. -->
      <div class="sticky bottom-0 z-10 flex items-center justify-end gap-3 border-t border-slate-200 bg-slate-50/95 py-4 backdrop-blur">
        <p v-if="problem" class="mr-auto text-sm text-amber-700">{{ problem }}</p>
        <AppButton variant="secondary" :disabled="saving" @click="goBack">Cancel</AppButton>
        <AppButton type="submit" :disabled="saving || !!problem">
          {{ saving ? 'Saving…' : sessionId ? 'Save' : 'Create' }}
        </AppButton>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { ArrowLeftIcon, TrashIcon } from '@heroicons/vue/24/outline'
import { ApiError } from '../../api'
import AppButton from '../../components/button/AppButton.vue'
import SessionEditor from '../../components/training/SessionEditor.vue'
import { newSession, sessionProblem, toSessionInput, toSessionRow, type SessionRow } from '../../components/training/trainingForm'
import { confirmDialog } from '../../confirm'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import { useSessionsStore } from '../../stores/sessions'

const router = useRouter()
const route = useRoute()
const store = useSessionsStore()
const exercisesStore = useExercisesStore()
const lookups = useLookupsStore()
// Set when editing a template (route /sessions/:id), absent when creating one.
const sessionId = route.params.id as string | undefined

// Null while an existing template loads.
const session = ref<SessionRow | null>(sessionId ? null : newSession(''))
const saving = ref(false)

// Why the session can't be saved yet, if anything; mirrors the server's checks.
const problem = computed(() => (session.value ? sessionProblem(session.value) : ''))

onMounted(async () => {
  try {
    const [loaded] = await Promise.all([
      sessionId ? store.getSession(sessionId) : undefined,
      exercisesStore.loadExercises(),
      exercisesStore.loadRatings(),
      lookups.loadTrainingLookups(),
    ])
    if (loaded) session.value = toSessionRow(loaded, false)
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 404 ? 'Session not found.' : 'Could not load the session.')
  }
})

async function submit() {
  if (!session.value || problem.value) return
  const input = toSessionInput(session.value)
  saving.value = true
  try {
    if (sessionId) {
      // Stay on the page so further changes can follow.
      await store.updateSession(sessionId, input)
      toast.success('Session saved.')
    } else {
      await store.createSession(input)
      toast.success('Session created.')
      await goBack()
    }
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not save the session.')
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!sessionId) return
  const ok = await confirmDialog({
    title: `Delete the session "${session.value?.name}"?`,
    message: 'It is removed from your library. Trainings keep their copies.',
    confirmLabel: 'Delete',
    danger: true,
  })
  if (!ok) return
  saving.value = true
  try {
    await store.deleteSession(sessionId)
    toast.success('Session deleted.')
    await goBack()
  } catch {
    toast.error('Could not delete the session.')
  } finally {
    saving.value = false
  }
}

function goBack() {
  return router.push({ name: 'sessions' })
}
</script>
