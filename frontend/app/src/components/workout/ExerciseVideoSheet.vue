<template>
  <TransitionRoot :show="open" as="template">
    <Dialog class="relative z-50" @close="emit('close')">
      <TransitionChild
        as="template"
        enter="duration-150 ease-out"
        enter-from="opacity-0"
        leave="duration-100 ease-in"
        leave-to="opacity-0"
      >
        <div class="fixed inset-0 bg-slate-900/40 transition-opacity" aria-hidden="true" />
      </TransitionChild>

      <!-- A bottom sheet on phones, a centered panel on wider screens. -->
      <div class="fixed inset-0 flex items-end justify-center sm:items-center sm:p-4">
        <TransitionChild
          as="template"
          enter="duration-200 ease-out"
          enter-from="translate-y-full sm:translate-y-0 sm:scale-95 sm:opacity-0"
          leave="duration-150 ease-in"
          leave-to="translate-y-full sm:translate-y-0 sm:scale-95 sm:opacity-0"
        >
          <DialogPanel
            class="flex max-h-[85dvh] w-full max-w-md flex-col rounded-t-2xl bg-white shadow-2xl transition sm:rounded-2xl"
          >
            <div class="flex items-center gap-2 border-b border-slate-100 py-2 pr-2 pl-4">
              <DialogTitle class="min-w-0 flex-1 truncate font-semibold text-slate-900">{{ name }}</DialogTitle>
              <AppButton variant="ghost" size="touch" icon aria-label="Close" @click="emit('close')">
                <XMarkIcon class="size-6" aria-hidden="true" />
              </AppButton>
            </div>

            <ul class="space-y-4 overflow-y-auto p-4 pb-[calc(1rem+env(safe-area-inset-bottom))]">
              <li v-for="(video, i) in videos" :key="i">
                <p v-if="videos.length > 1" class="mb-1 text-xs font-medium text-slate-500">Video {{ i + 1 }} of {{ videos.length }}</p>
                <div v-if="video.embed" :class="{ 'flex justify-center': video.embed.provider === 'instagram' }">
                  <VideoPreview :url="video.url" />
                </div>
                <AppButton
                  :href="video.url"
                  :variant="video.embed ? 'text' : 'secondary'"
                  size="touch"
                  :class="video.embed ? 'mt-1 -ml-3' : 'w-full'"
                >
                  {{ video.embed ? `Open in ${providerNames[video.embed.provider]}` : `Open on ${video.host}` }}
                  <ArrowTopRightOnSquareIcon class="size-4" aria-hidden="true" />
                </AppButton>
              </li>
            </ul>
          </DialogPanel>
        </TransitionChild>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { ArrowTopRightOnSquareIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import AppButton from '../button/AppButton.vue'
import VideoPreview from '../video/VideoPreview.vue'
import { videoEmbed, type VideoProvider } from '../../videoEmbed'

// An exercise's videos in their saved order, played in place so the workout isn't left; each
// can also be opened in its app. Links that can't be embedded show as a button only.
const props = defineProps<{ open: boolean; name: string; urls: string[] }>()
const emit = defineEmits<{ close: [] }>()

const providerNames: Record<VideoProvider, string> = { youtube: 'YouTube', instagram: 'Instagram' }

const videos = computed(() =>
  props.urls
    // Only http(s) links, never e.g. javascript: URLs.
    .filter((url) => /^https?:\/\//i.test(url))
    .map((url) => ({ url, embed: videoEmbed(url), host: hostOf(url) })),
)

function hostOf(url: string) {
  try {
    return new URL(url).hostname.replace(/^www\./, '')
  } catch {
    return 'the web'
  }
}
</script>
