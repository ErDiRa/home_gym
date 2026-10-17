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
		<p v-if="!readonly" class="mb-6 text-sm text-slate-500">
			<span class="text-red-600">*</span> Required
		</p>
		<div v-else class="mb-6" />

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
				<textarea
					v-if="field.multiline"
					v-model="form[field.key]"
					rows="3"
					:disabled="readonly"
					:class="inputClass"
				/>
				<input
					v-else
					v-model="form[field.key]"
					:type="field.type ?? 'text'"
					:required="field.required"
					:disabled="readonly"
					:placeholder="field.placeholder"
					:class="inputClass"
				/>
			</label>

			<fieldset>
				<legend class="text-sm font-medium text-slate-800">
					Type<span v-if="!readonly" class="text-red-600" aria-hidden="true"> *</span>
				</legend>
				<div class="mt-1 inline-flex rounded-md border border-slate-300 bg-white p-0.5">
					<label
						v-for="(label, type) in exerciseTypeLabels"
						:key="type"
						:class="[
							'rounded px-4 py-1.5 text-sm font-medium has-focus-visible:ring-2 has-focus-visible:ring-indigo-200',
							form.type === type ? 'bg-indigo-600 text-white' : 'text-slate-700',
							readonly ? 'cursor-default' : form.type === type ? 'cursor-pointer' : 'cursor-pointer hover:bg-slate-100'
						]"
					>
						<input
							v-model="form.type"
							type="radio"
							name="type"
							:value="type"
							:disabled="readonly"
							required
							class="sr-only"
						/>
						{{ label }}
					</label>
				</div>
			</fieldset>

			<fieldset v-if="!readonly || hasVideos">
				<legend class="text-sm font-medium text-slate-800">Video links</legend>
				<div v-for="(_, i) in form.video_urls" :key="i" class="mt-1 space-y-2">
					<div class="flex items-center gap-2">
						<input
							ref="videoInputs"
							v-model="form.video_urls[i]"
							type="url"
							:disabled="readonly"
							placeholder="https://www.youtube.com/watch?v=…"
							:aria-label="`Video link ${i + 1}`"
							:class="[inputClass, 'mt-0!']"
						/>
						<button
							v-if="!readonly && form.video_urls.length > 1"
							type="button"
							:aria-label="`Remove video link ${i + 1}`"
							class="rounded-md p-2 text-slate-400 hover:bg-slate-100 hover:text-red-600"
							@click="form.video_urls.splice(i, 1)"
						>
							<XMarkIcon class="size-4" aria-hidden="true" />
						</button>
					</div>
					<VideoPreview :url="form.video_urls[i]" />
				</div>
				<button
					v-if="!readonly && form.video_urls.length < MAX_VIDEOS"
					type="button"
					class="mt-2 inline-flex items-center gap-x-1 rounded-md px-1 py-0.5 text-sm font-medium text-indigo-600 hover:text-indigo-800 focus-visible:ring-2 focus-visible:ring-indigo-200 focus-visible:outline-none"
					@click="addVideo"
				>
					<PlusIcon class="size-4" aria-hidden="true" />
					Add video link
				</button>
			</fieldset>

			<div>
				<MultiSelect
					v-model="form.focus_ids"
					label="Focus"
					placeholder="Select focuses (e.g. weightlifting, shoulder health)"
					:disabled="readonly"
					:options="focuses"
				/>
				<div v-if="!readonly" class="mt-2">
					<AppButton
						v-if="!addingFocusOpen"
						variant="secondary"
						size="md"
						@click="addingFocusOpen = true"
					>
						<span class="inline-flex items-center gap-x-1">
							<PlusIcon class="size-4" aria-hidden="true" />
							Add focus
						</span>
					</AppButton>
					<div v-else class="flex flex-wrap items-center gap-2">
						<input
							ref="newFocusInput"
							v-model="newFocus"
							type="text"
							maxlength="60"
							placeholder="New focus"
							@keydown.enter.prevent="addFocus"
							@keydown.esc.prevent="closeAddFocus"
							:class="[inputClass, 'mt-0! max-w-56 py-1.5!']"
						/>
						<input
							v-model="newAlias"
							type="text"
							maxlength="10"
							placeholder="Alias"
							title="Short label shown as a tag in the table"
							@keydown.enter.prevent="addFocus"
							@keydown.esc.prevent="closeAddFocus"
							:class="[inputClass, 'mt-0! max-w-28 py-1.5!']"
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
				:disabled="readonly"
				:options="joints"
			/>
			<MultiSelect
				v-model="form.muscle_group_ids"
				label="Muscle groups"
				placeholder="Select muscle groups"
				required
				:disabled="readonly"
				:options="muscleGroups"
			/>

			<div v-if="!readonly" class="flex justify-end gap-3 pt-2">
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
	import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { toast } from 'vue-sonner';
	import {
		ApiError,
		createExercise,
		createFocus,
		getExercise,
		listFocuses,
		listJoints,
		listMuscleGroups,
		updateExercise
	} from '../../api';
	import AppButton from '../../components/button/AppButton.vue';
	import MultiSelect from '../../components/form/MultiSelect.vue';
	import VideoPreview from '../../components/video/VideoPreview.vue';
	import { state } from '../../session';
	import {
		exerciseTypeLabels,
		type ExerciseType,
		type Focus,
		type Option
	} from '../../types';

	const router = useRouter();
	// Set when editing an existing exercise (route /exercises/:id), absent when creating one.
	const exerciseId = useRoute().params.id as string | undefined;
	// Only admins can change exercises; everyone else gets the filled form read-only.
	const readonly = computed(() => !state.user?.is_admin);
	const title = computed(() =>
		readonly.value ? 'Exercise' : exerciseId ? 'Edit exercise' : 'New exercise'
	);

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
	const inputClass =
		'mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-normal outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200';

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
	const joints = ref<Option[]>([]);
	const muscleGroups = ref<Option[]>([]);
	const focuses = ref<Focus[]>([]);
	const newFocus = ref('');
	const newAlias = ref('');
	const addingFocusOpen = ref(false);
	const newFocusInput = ref<HTMLInputElement | null>(null);

	watch(addingFocusOpen, async (open) => {
		if (open) {
			await nextTick();
			newFocusInput.value?.focus();
		}
	});
	const addingFocus = ref(false);

	// The server allows at most this many video links per exercise.
	const MAX_VIDEOS = 10;
	const videoInputs = ref<HTMLInputElement[]>([]);
	const hasVideos = computed(() => form.video_urls.some((u) => u.trim()));

	// Shows another link input and focuses it.
	async function addVideo() {
		form.video_urls.push('');
		await nextTick();
		videoInputs.value[form.video_urls.length - 1]?.focus();
	}
	const valid = computed(
		() =>
			form.type !== '' &&
			form.joint_ids.length > 0 &&
			form.muscle_group_ids.length > 0
	);

	onMounted(async () => {
		try {
			const [loadedJoints, loadedMuscleGroups, loadedFocuses, exercise] =
				await Promise.all([
					listJoints(),
					listMuscleGroups(),
					listFocuses(),
					exerciseId ? getExercise(exerciseId) : undefined
				]);
			joints.value = loadedJoints;
			muscleGroups.value = loadedMuscleGroups;
			focuses.value = loadedFocuses;
			if (exercise) {
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
			const focus = await createFocus(
				newFocus.value.trim(),
				newAlias.value.trim()
			);
			focuses.value = [...focuses.value, focus].sort((a, b) =>
				a.name.localeCompare(b.name)
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
			type: form.type,
			video_urls: form.video_urls.map((u) => u.trim()).filter(Boolean)
		};
		saving.value = true;
		try {
			await (exerciseId
				? updateExercise(exerciseId, input)
				: createExercise(input));
			toast.success(exerciseId ? 'Exercise saved.' : 'Exercise created.');
			await router.push({ name: 'exercises' });
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
