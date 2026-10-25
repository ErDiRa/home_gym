<template>
  <section>
    <h1 class="text-2xl font-bold tracking-tight text-slate-900">Settings</h1>

    <section class="mt-6 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm" aria-label="WHOOP">
      <h2 class="font-semibold text-slate-900">WHOOP</h2>
      <p v-if="!whoop.status" class="mt-1 text-sm text-slate-500">Loading…</p>
      <p v-else-if="!whoop.status.configured" class="mt-1 text-sm text-slate-500">
        WHOOP is not set up on this server (WHOOP_CLIENT_ID, WHOOP_CLIENT_SECRET).
      </p>
      <template v-else>
        <p class="mt-1 text-sm text-slate-600">
          {{
            whoop.status.connected
              ? 'Connected. Finished workouts can retrieve their WHOOP activity.'
              : 'Connect your WHOOP account to add strain, heart rate and recovery to finished workouts.'
          }}
        </p>
        <AppButton v-if="whoop.status.connected" variant="secondary" size="touch" class="mt-3 w-full" @click="disconnect">
          Disconnect
        </AppButton>
        <AppButton v-else size="touch" class="mt-3 w-full" @click="whoop.connectWhoop">Connect WHOOP</AppButton>
      </template>
    </section>
  </section>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import AppButton from '../../components/button/AppButton.vue'
import { confirmDialog } from '../../confirm'
import { useWhoopStore } from '../../stores/whoop'

const route = useRoute()
const router = useRouter()
const whoop = useWhoopStore()

onMounted(async () => {
  // Back from WHOOP: the callback adds ?whoop=error when connecting failed.
  if (route.query.whoop === 'error') {
    toast.error('Could not connect WHOOP. Please try again.')
    router.replace({ query: {} })
  }
  await whoop.loadStatus().catch(() => toast.error('Could not load the WHOOP connection.'))
})

async function disconnect() {
  const ok = await confirmDialog({
    title: 'Disconnect WHOOP?',
    message: 'Workouts keep the WHOOP data they already have.',
    confirmLabel: 'Disconnect',
    danger: true,
  })
  if (!ok) return
  try {
    await whoop.disconnectWhoop()
    await whoop.loadStatus()
  } catch {
    toast.error('Could not disconnect WHOOP.')
  }
}
</script>
