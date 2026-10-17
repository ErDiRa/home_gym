<template>
	<!-- Sandboxed: the players need scripts and their own origin, but can't navigate this page. -->
	<div
		v-if="embed"
		:class="embed.provider === 'youtube' ? 'aspect-video' : 'h-136 max-w-sm'"
		class="w-full"
	>
		<iframe
			:src="embed.src"
			:title="
				embed.provider === 'youtube'
					? 'YouTube video preview'
					: 'Instagram video preview'
			"
			class="size-full rounded-lg border border-slate-200 bg-slate-50"
			loading="lazy"
			referrerpolicy="strict-origin-when-cross-origin"
			sandbox="allow-scripts allow-same-origin allow-presentation allow-popups allow-popups-to-escape-sandbox"
			allow="encrypted-media; picture-in-picture; fullscreen"
			allowfullscreen
		/>
	</div>
</template>

<script setup lang="ts">
	import { computed } from 'vue';
	import { videoEmbed } from '../../videoEmbed';

	const props = defineProps<{ url: string }>();
	// Only YouTube and Instagram links are previewed; anything else shows nothing.
	const embed = computed(() => videoEmbed(props.url));
</script>
