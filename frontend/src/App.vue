<template>
  <RouterView />
  <!-- One toaster for the whole app; anything can call toast() from vue-sonner. -->
  <Toaster position="top-right" rich-colors close-button />
</template>

<script setup lang="ts">
import { Toaster } from 'vue-sonner'
import 'vue-sonner/style.css'
</script>
