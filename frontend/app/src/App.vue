<template>
  <RouterView />
  <!-- One confirmation modal, opened by confirmDialog() from confirm.ts. -->
  <ConfirmDialog />
  <!-- One toaster for the whole app; anything can call toast() from vue-sonner. -->
  <Toaster position="top-center" rich-colors close-button />
</template>

<script setup lang="ts">
import { Toaster } from 'vue-sonner'
import ConfirmDialog from './components/ui/ConfirmDialog.vue'
import 'vue-sonner/style.css'
</script>
