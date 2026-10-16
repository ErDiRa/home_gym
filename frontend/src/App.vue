<template>
  <RouterView />
  <Toaster position="top-right" rich-colors close-button />
</template>

<script setup lang="ts">
import { Toaster } from 'vue-sonner'
import 'vue-sonner/style.css'
</script>
