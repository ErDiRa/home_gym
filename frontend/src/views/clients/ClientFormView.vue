<template>
  <div class="max-w-2xl">
    <RouterLink
      :to="{ name: 'clients' }"
      class="mb-4 inline-flex items-center gap-x-1.5 text-sm font-medium text-slate-600 hover:text-indigo-600"
    >
      <ArrowLeftIcon class="size-4" aria-hidden="true" />
      Back to clients
    </RouterLink>

    <div class="mb-1 flex items-center justify-between gap-4">
      <h1 class="text-xl font-bold text-slate-900">{{ clientId ? 'Edit client' : 'New client' }}</h1>
      <AppButton v-if="clientId" variant="secondary" size="md" :disabled="saving" @click="remove">
        <TrashIcon class="size-4" aria-hidden="true" />
        Delete client
      </AppButton>
    </div>
    <p class="mb-6 text-sm text-slate-500"><span class="text-red-600">*</span> Required</p>

    <form class="space-y-4" @submit.prevent="submit">
      <div class="grid gap-4 sm:grid-cols-2">
        <label class="block text-sm font-medium text-slate-800">
          First name<span class="text-red-600" aria-hidden="true"> *</span>
          <input v-model="form.first_name" type="text" required maxlength="100" autocomplete="off" :class="inputClass" />
        </label>
        <label class="block text-sm font-medium text-slate-800">
          Last name<span class="text-red-600" aria-hidden="true"> *</span>
          <input v-model="form.last_name" type="text" required maxlength="100" autocomplete="off" :class="inputClass" />
        </label>
      </div>
      <label class="block text-sm font-medium text-slate-800">
        Email<span class="text-red-600" aria-hidden="true"> *</span>
        <input v-model="form.email" type="email" required autocomplete="off" :class="inputClass" />
      </label>
      <label v-if="!clientId" class="block text-sm font-medium text-slate-800">
        Initial password<span class="text-red-600" aria-hidden="true"> *</span>
        <input
          v-model="password"
          type="text"
          required
          minlength="8"
          maxlength="72"
          autocomplete="new-password"
          :class="inputClass"
        />
        <span class="mt-1 block text-xs font-normal text-slate-500">
          At least 8 characters. Shown in plain text so you can pass it on to the client; it can't be changed later.
        </span>
      </label>

      <div class="flex justify-end gap-3 pt-2">
        <AppButton variant="secondary" :disabled="saving" @click="router.push({ name: 'clients' })">Cancel</AppButton>
        <AppButton type="submit" :disabled="saving">
          {{ saving ? 'Saving…' : clientId ? 'Save' : 'Create' }}
        </AppButton>
      </div>
    </form>

    <section v-if="clientId" class="mt-10">
      <div class="mb-2 flex items-center justify-between gap-4">
        <h2 class="text-sm font-medium text-slate-800">Trainings</h2>
        <AppButton size="sm" :to="{ name: 'training-new', query: { client: clientId } }">
          <PlusIcon class="size-4" aria-hidden="true" />
          New training
        </AppButton>
      </div>
      <p
        v-if="!trainings.length"
        class="rounded-lg border border-dashed border-slate-300 px-4 py-6 text-center text-sm text-slate-500"
      >
        No trainings yet.
      </p>
      <ul v-else class="divide-y divide-slate-200 rounded-lg border border-slate-200 bg-white">
        <li v-for="training in trainings" :key="training.id">
          <RouterLink
            :to="{ name: 'training-edit', params: { id: training.id }, query: { client: clientId } }"
            class="flex items-center gap-3 px-4 py-3 hover:bg-slate-50"
          >
            <span class="min-w-0 flex-1 truncate text-sm font-medium text-slate-900">{{ training.name }}</span>
            <ActiveBadge v-if="training.active" />
            <span class="text-xs text-slate-400">
              {{ training.unit_count }} {{ training.unit_count === 1 ? 'unit' : 'units' }} · updated
              {{ new Date(training.updated_at).toLocaleDateString() }}
            </span>
          </RouterLink>
        </li>
      </ul>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { ArrowLeftIcon, PlusIcon, TrashIcon } from '@heroicons/vue/24/outline'
import { ApiError, createClient, deleteClient, getClient, updateClient } from '../../api'
import AppButton from '../../components/button/AppButton.vue'
import ActiveBadge from '../../components/training/ActiveBadge.vue'
import type { ClientInput, ClientTraining } from '../../types'

const router = useRouter()
// Set when editing an existing client (route /clients/:id), absent when creating one.
const clientId = useRoute().params.id as string | undefined

const inputClass =
  'mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-normal outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200'

const form = reactive<ClientInput>({ first_name: '', last_name: '', email: '' })
const password = ref('')
const trainings = ref<ClientTraining[]>([])
const saving = ref(false)

onMounted(async () => {
  if (!clientId) return
  try {
    const client = await getClient(clientId)
    Object.assign(form, { first_name: client.first_name, last_name: client.last_name, email: client.email })
    trainings.value = client.trainings
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 404 ? 'Client not found.' : 'Could not load the client.')
  }
})

async function submit() {
  saving.value = true
  try {
    if (clientId) {
      await updateClient(clientId, form)
      toast.success('Client saved.')
      await router.push({ name: 'clients' })
    } else {
      const client = await createClient({ ...form, password: password.value })
      toast.success(`Client ${client.first_name} ${client.last_name} created.`)
      // Continue on the edit page, where the client's trainings are created.
      await router.push({ name: 'client-edit', params: { id: client.id } })
    }
  } catch (e) {
    toast.error(e instanceof ApiError && [400, 409].includes(e.status) ? e.message : 'Could not save the client.')
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!clientId) return
  const name = `${form.first_name} ${form.last_name}`
  if (!confirm(`Delete ${name}? They will no longer be able to sign in, and their trainings are hidden.`)) return
  saving.value = true
  try {
    await deleteClient(clientId)
    toast.success(`${name} deleted.`)
    await router.push({ name: 'clients' })
  } catch {
    toast.error('Could not delete the client.')
  } finally {
    saving.value = false
  }
}
</script>
