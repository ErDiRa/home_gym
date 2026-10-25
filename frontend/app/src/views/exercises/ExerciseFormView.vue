<template>
	<div class="max-w-2xl">
		<RouterLink
			:to="{ name: 'exercises' }"
			class="mb-4 inline-flex items-center gap-x-1.5 text-sm font-medium text-slate-600 hover:text-indigo-600"
		>
			<ArrowLeftIcon class="size-4" aria-hidden="true" />
			Back to exercises
		</RouterLink>

		<h1 class="mb-1 text-xl font-bold text-slate-900">{{ title }}</h1>
		<p class="mb-6 text-sm text-slate-500">
			<span class="text-red-600">*</span> Required
		</p>

		<form class="space-y-4" @submit.prevent="submit">
			<label
				v-for="field in fields"
				:key="field.key"
				class="block text-sm font-medium text-slate-800"
			>
				{{ field.label
				}}<span v-if="field.required" class="text-red-600" aria-hidden="true">
					*</span
				>
				<TextInput
					v-model="form[field.key]"
					:multiline="field.multiline"
					:rows="field.multiline ? 3 : undefined"
					:type="field.multiline ? undefined : (field.type ?? 'text')"
					:required="field.required"
					:placeholder="field.placeholder"
					class="mt-1"
				/>
			</label>

			<div>
				<p class="text-sm font-medium text-slate-800" aria-hidden="true">
					Type<span class="text-red-600"> *</span>
				</p>
				<SegmentedControl v-model="form.type" :options="typeOptions" label="Type" class="mt-1" />
			</div>

			<fieldset>
				<legend class="text-sm font-medium text-slate-800">Video links</legend>
				<div v-for="(_, i) in form.video_urls" :key="i" class="mt-1 space-y-2">
					<div class="flex items-center gap-2">
						<TextInput
							ref="videoInputs"
							v-model="form.video_urls[i]"
							type="url"
							placeholder="https://www.youtube.com/watch?v=…"
							:aria-label="`Video link ${i + 1}`"
						/>
						<AppButton
							v-if="form.video_urls.length > 1"
							variant="ghost"
							size="md"
							icon
							class="hover:text-red-600"
							:aria-label="`Remove video link ${i + 1}`"
							@click="form.video_urls.splice(i, 1)"
						>
							<XMarkIcon class="size-4" aria-hidden="true" />
						</AppButton>
					</div>
					<VideoPreview :url="form.video_urls[i]" />
				</div>
				<AppButton v-if="form.video_urls.length < MAX_VIDEOS" variant="text" size="sm" class="mt-2 -ml-2.5" @click="addVideo">
					<PlusIcon class="size-4" aria-hidden="true" />
					Add video link
				</AppButton>
			</fieldset>

			<div>
				<MultiSelect
					v-model="form.focus_ids"
					label="Focus"
					placeholder="Select focuses (e.g. weightlifting, shoulder health)"
					:options="focuses"
				/>
				<div class="mt-2">
					<AppButton
						v-if="!addingFocusOpen"
						variant="secondary"
						size="md"
						@click="addingFocusOpen = true"
					>
						<PlusIcon class="size-4" aria-hidden="true" />
						Add focus
					</AppButton>
					<div v-else class="flex flex-wrap items-center gap-2">
						<TextInput
							ref="newFocusInput"
							v-model="newFocus"
							type="text"
							maxlength="60"
							placeholder="New focus"
							class="h-9! max-w-56"
							@keydown.enter.prevent="addFocus"
							@keydown.esc.prevent="closeAddFocus"
						/>
						<TextInput
							v-model="newAlias"
							type="text"
							maxlength="10"
							placeholder="Alias"
							title="Short label shown as a tag in the table"
							class="h-9! max-w-28"
							@keydown.enter.prevent="addFocus"
							@keydown.esc.prevent="closeAddFocus"
						/>
						<AppButton
							size="md"
							:disabled="addingFocus || !newFocus.trim() || !newAlias.trim()"
							@click="addFocus"
						>
							Add
						</AppButton>
						<AppButton
							variant="secondary"
							size="md"
							:disabled="addingFocus"
							@click="closeAddFocus"
							>Cancel</AppButton
						>
					</div>
				</div>
			</div>

			<MultiSelect
				v-model="form.joint_ids"
				label="Joints"
				placeholder="Select joints"
				required
				:options="joints"
			/>
			<MultiSelect
				v-model="form.muscle_group_ids"
				label="Muscle groups"
				placeholder="Select muscle groups"
				required
				:options="muscleGroups"
			/>

			<!-- The user's own rating: how helpful the exercise is for them. Only for a saved one. -->
			<fieldset v-if="exerciseId">
				<legend class="text-sm font-medium text-slate-800">Your rating</legend>
				<p class="text-xs text-slate-500">How helpful is it for you? Only you see it.</p>
				<StarRating v-model="rating" label="Your rating" size="sm" class="mt-1 -ml-1" />
				<TextInput
					v-model="ratingNotes"
					multiline
					rows="2"
					maxlength="2000"
					placeholder="Rating note, e.g. Knees feel good"
					class="mt-2"
					aria-label="Rating note"
				/>
			</fieldset>

			<!-- Sticks to the bottom of the screen so saving doesn't need scrolling down. -->
			<div
				class="sticky bottom-0 z-10 flex justify-end gap-3 border-t border-slate-200 bg-slate-50/95 py-4 backdrop-blur"
			>
				<AppButton
					variant="secondary"
					:disabled="saving"
					@click="router.push({ name: 'exercises' })"
				>
					Cancel
				</AppButton>
				<AppButton type="submit" :disabled="saving || !valid">{{
					saving ? 'Saving…' : exerciseId ? 'Save' : 'Create'
				}}</AppButton>
			</div>
		</form>
	</div>
</template>

<script setup lang="ts">
	import { ArrowLeftIcon, PlusIcon, XMarkIcon } from '@heroicons/vue/24/outline';
	import { storeToRefs } from 'pinia';
	import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { toast } from 'vue-sonner';
	import { ApiError } from '../../api';
	import AppButton from '../../components/button/AppButton.vue';
	import StarRating from '../../components/exercise/StarRating.vue';
	import MultiSelect from '../../components/form/MultiSelect.vue';
	import SegmentedControl from '../../components/form/SegmentedControl.vue';
	import TextInput, { type TextInputExposed } from '../../components/form/TextInput.vue';
	import VideoPreview from '../../components/video/VideoPreview.vue';
	import { useExercisesStore } from '../../stores/exercises';
	import { useLookupsStore } from '../../stores/lookups';
	import { exerciseTypeLabels, type ExerciseType } from '../../types';

	const router = useRouter();
	const exercisesStore = useExercisesStore();
	const lookups = useLookupsStore();
	const { joints, muscleGroups, focuses } = storeToRefs(lookups);
	// Set when editing an existing exercise (route /exercises/:id), absent when creating one.
	const exerciseId = useRoute().params.id as string | undefined;
	const title = exerciseId ? 'Edit exercise' : 'New exercise';

	type FormKey = 'name' | 'description';
	const fields: {
		key: FormKey;
		label: string;
		required?: boolean;
		multiline?: boolean;
		type?: string;
		placeholder?: string;
	}[] = [
		{ key: 'name', label: 'Name', required: true },
		{ key: 'description', label: 'Description', multiline: true }
	];
	const typeOptions = Object.entries(exerciseTypeLabels).map(([value, label]) => ({
		value: value as ExerciseType,
		label
	}));

	const form = reactive({
		name: '',
		// Empty until picked, so a new exercise needs an explicit choice.
		type: '' as ExerciseType | '',
		description: '',
		// Always at least one (possibly empty) input; blanks are dropped on save.
		video_urls: [''] as string[],
		joint_ids: [] as string[],
		muscle_group_ids: [] as string[],
		focus_ids: [] as string[]
	});
	const saving = ref(false);
	// The user's rating of the exercise (0 = not rated) and its note, saved with the exercise.
	const rating = ref(0);
	const ratingNotes = ref('');
	const newFocus = ref('');
	const newAlias = ref('');
	const addingFocusOpen = ref(false);
	const newFocusInput = ref<TextInputExposed | null>(null);

	// Focus the new-focus input once it has rendered.
	watch(addingFocusOpen, async (open) => {
		if (open) {
			await nextTick();
			newFocusInput.value?.focus();
		}
	});
	// True while the new focus is being saved (addingFocusOpen is whether its fields show).
	const addingFocus = ref(false);

	// The server allows at most this many video links per exercise.
	const MAX_VIDEOS = 10;
	// Filled by the template's ref inside v-for: one element per link input, in order.
	const videoInputs = ref<TextInputExposed[]>([]);

	// Shows another link input and focuses it.
	async function addVideo() {
		form.video_urls.push('');
		await nextTick();
		videoInputs.value[form.video_urls.length - 1]?.focus();
	}
	// The fields the server requires besides the name (which the input's `required` covers).
	const valid = computed(
		() =>
			form.type !== '' &&
			form.joint_ids.length > 0 &&
			form.muscle_group_ids.length > 0
	);

	onMounted(async () => {
		try {
			const [exercise] = await Promise.all([
				exerciseId ? exercisesStore.getExercise(exerciseId) : undefined,
				lookups.loadExerciseLookups(),
				exerciseId ? exercisesStore.loadRatings() : undefined
			]);
			const saved = exerciseId ? exercisesStore.ratings.get(exerciseId) : undefined;
			rating.value = saved?.rating ?? 0;
			ratingNotes.value = saved?.notes ?? '';
			if (exercise) {
				// The form holds everything but the id, which comes from the route.
				const { id: _id, ...fields } = exercise;
				Object.assign(form, fields);
				if (!form.video_urls.length) form.video_urls = [''];
			}
		} catch (e) {
			toast.error(
				e instanceof ApiError && e.status === 404
					? 'Exercise not found.'
					: 'Could not load the exercise.'
			);
		}
	});

	// Hides the add-focus fields and clears them.
	function closeAddFocus() {
		addingFocusOpen.value = false;
		newFocus.value = '';
		newAlias.value = '';
	}

	// Creates a new focus and selects it for this exercise right away.
	async function addFocus() {
		if (addingFocus.value || !newFocus.value.trim() || !newAlias.value.trim())
			return;
		addingFocus.value = true;
		try {
			const focus = await lookups.createFocus(
				newFocus.value.trim(),
				newAlias.value.trim()
			);
			form.focus_ids.push(focus.id);
			closeAddFocus();
			toast.success(`Focus "${focus.name}" added.`);
		} catch (e) {
			toast.error(
				e instanceof ApiError && [400, 409].includes(e.status)
					? e.message
					: 'Could not add the focus.'
			);
		} finally {
			addingFocus.value = false;
		}
	}

	async function submit() {
		if (form.type === '') return;
		const input = {
			...form,
			type: form.type, // narrowed to ExerciseType by the check above
			video_urls: form.video_urls.map((u) => u.trim()).filter(Boolean)
		};
		saving.value = true;
		try {
			if (exerciseId) {
				// Stay on the page so further changes can follow.
				await exercisesStore.updateExercise(exerciseId, input);
				// A rating is saved when one is set and it or its note changed.
				const saved = exercisesStore.ratings.get(exerciseId);
				const notes = ratingNotes.value.trim();
				if (rating.value && (rating.value !== saved?.rating || notes !== (saved?.notes ?? ''))) {
					await exercisesStore.rateExercise(exerciseId, rating.value, notes);
				}
				toast.success('Exercise saved.');
			} else {
				await exercisesStore.createExercise(input);
				toast.success('Exercise created.');
				await router.push({ name: 'exercises' });
			}
		} catch (e) {
			toast.error(
				e instanceof ApiError && e.status === 400
					? e.message
					: 'Could not save the exercise.'
			);
		} finally {
			saving.value = false;
		}
	}
</script>
