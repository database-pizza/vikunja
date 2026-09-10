<template>
	<div
		:class="{ 'is-loading': taskService.loading }"
		class="defer-task loading-container"
		@click.stop
		@mousedown.stop
		@pointerdown.stop
		@touchstart.stop
	>
		<label class="label">{{ $t('task.deferDueDate.title') }}</label>
		<div class="defer-days">
			<XButton
				:shadow="false"
				variant="secondary"
				@click.prevent.stop="() => deferDays(1)"
			>
				{{ $t('task.deferDueDate.1day') }}
			</XButton>
			<XButton
				:shadow="false"
				variant="secondary"
				@click.prevent.stop="() => deferDays(3)"
			>
				{{ $t('task.deferDueDate.3days') }}
			</XButton>
			<XButton
				:shadow="false"
				variant="secondary"
				@click.prevent.stop="() => deferDays(7)"
			>
				{{ $t('task.deferDueDate.1week') }}
			</XButton>
		</div>
		<DatepickerInline
			v-model="dueDate"
			:show-shortcuts="false"
			@update:modelValue="updateDueDate"
		/>
	</div>
</template>

<script setup lang="ts">
import {ref, shallowReactive, watch} from 'vue'

import DatepickerInline from '@/components/input/DatepickerInline.vue'

import TaskService from '@/services/task'
import type {ITask} from '@/modelTypes/ITask'

const props = defineProps<{
	modelValue: ITask,
}>()

const emit = defineEmits<{
	'update:modelValue': [value: ITask]
}>()

const taskService = shallowReactive(new TaskService())
const task = ref<ITask>()

// We're saving the due date separately to prevent null errors in very short periods where the task is null.
const dueDate = ref<Date | null>(null)
const lastValue = ref<Date | null>()

watch(
	() => props.modelValue,
	(value) => {
		task.value = { ...value }
		dueDate.value = value.dueDate
		lastValue.value = value.dueDate
	},
	{immediate: true},
)

function deferDays(days: number) {
	const deferred = new Date(dueDate.value ?? new Date())
	deferred.setDate(deferred.getDate() + days)
	dueDate.value = deferred
	updateDueDate()
}

async function updateDueDate() {
	if (!dueDate.value) {
		return
	}

	if (+new Date(dueDate.value) === +lastValue.value) {
		return
	}

	const newTask = await taskService.update({
		...task.value,
		dueDate: new Date(dueDate.value),
	})
	lastValue.value = newTask.dueDate
	task.value = newTask
	emit('update:modelValue', newTask)
}
</script>

<style lang="scss" scoped>
// 100px is roughly the size the pane is pulled to the right
$defer-task-max-width: 350px + 100px;

.defer-task {
	inline-size: 100%;
	max-inline-size: $defer-task-max-width;

	@media screen and (max-width: ($defer-task-max-width)) {
		inset-inline-start: .5rem;
		inset-inline-end: .5rem;
		max-inline-size: 100%;
		inline-size: calc(100vw - 1rem - 2rem);
	}
}

.defer-days {
	justify-content: space-between;
	display: flex;
	margin: .5rem 0;
}
</style>
