<template>
	<div
		ref="root"
		class="calendar-month"
		:class="{'is-large': large}"
	>
		<div class="calendar-month__nav">
			<BaseButton
				v-tooltip="$t('input.datepicker.previousMonth')"
				class="calendar-month__nav-button"
				:aria-label="$t('input.datepicker.previousMonth')"
				@click.stop="moveMonth(-1)"
			>
				<Icon icon="angle-left" />
			</BaseButton>
			<div class="calendar-month__title">
				<select
					:value="view.month"
					class="calendar-month__month-select"
					:aria-label="$t('input.datepicker.month')"
					@change="setMonth(($event.target as HTMLSelectElement).value)"
				>
					<option
						v-for="(name, index) in monthNames"
						:key="index"
						:value="index"
					>
						{{ name }}
					</option>
				</select>
				<input
					:value="view.year"
					class="calendar-month__year-input"
					type="text"
					inputmode="numeric"
					pattern="[0-9]*"
					maxlength="4"
					:aria-label="$t('input.datepicker.year')"
					@change="setYear($event)"
					@keydown.enter.prevent="setYear($event)"
				>
			</div>
			<BaseButton
				v-tooltip="$t('input.datepicker.nextMonth')"
				class="calendar-month__nav-button"
				:aria-label="$t('input.datepicker.nextMonth')"
				@click.stop="moveMonth(1)"
			>
				<Icon icon="angle-right" />
			</BaseButton>
		</div>

		<div
			class="calendar-month__grid"
			role="grid"
			@mouseleave="hovered = null"
			@keydown="onKeydown"
		>
			<div
				v-for="label in weekdayLabels"
				:key="label"
				class="calendar-month__weekday"
				role="columnheader"
			>
				{{ label }}
			</div>
			<div
				v-for="cell in cells"
				:key="cell.date.getTime()"
				class="calendar-month__cell"
				:class="cell.classes"
				role="gridcell"
				@mouseenter="mode === 'range' && (hovered = cell.date)"
			>
				<button
					type="button"
					class="calendar-month__day"
					:class="{'is-today': isSameDay(cell.date, today)}"
					:tabindex="isSameDay(cell.date, focusedDate) ? 0 : -1"
					:aria-label="formatDate(cell.date, 'LL')"
					:aria-pressed="cell.classes['is-selected']"
					:disabled="isDisabled(cell.date) || undefined"
					:data-date="dateKey(cell.date)"
					@click.stop="pick(cell.date)"
					@focus="focusedDate = cell.date"
				>
					{{ cell.date.getDate() }}
				</button>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import {computed, nextTick, ref, watch} from 'vue'

import BaseButton from '@/components/base/BaseButton.vue'

import {useAuthStore} from '@/stores/auth'
import {formatDate} from '@/helpers/time/formatDate'
import {
	addDays,
	buildMonthGrid,
	isDayBetween,
	isSameDay,
	startOfDay,
	weekdayOrder,
	type CalendarCell,
} from '@/helpers/time/calendarGrid'

const props = withDefaults(defineProps<{
	mode?: 'single' | 'range'
	selected?: Date | null
	rangeStart?: Date | null
	rangeEnd?: Date | null
	minDate?: Date | null
	large?: boolean
}>(), {
	mode: 'single',
	selected: null,
	rangeStart: null,
	rangeEnd: null,
	minDate: null,
	large: false,
})

const emit = defineEmits<{
	pick: [date: Date]
}>()

const authStore = useAuthStore()
const weekStart = computed(() => authStore.settings.weekStart ?? 0)

const today = startOfDay(new Date())
const root = ref<HTMLElement | null>(null)
const hovered = ref<Date | null>(null)

const anchor = computed(() => props.mode === 'range' ? props.rangeStart : props.selected)

const initialDate = (props.mode === 'range' ? props.rangeStart : props.selected) ?? today
const view = ref(monthOf(initialDate))
const focusedDate = ref<Date>(initialDate)

function monthOf(date: Date) {
	return {year: date.getFullYear(), month: date.getMonth()}
}

const cells = computed(() => buildMonthGrid(view.value.year, view.value.month, weekStart.value)
	.map(cell => ({...cell, classes: cellClasses(cell)})))

// Follow outside changes (shortcut, v-model) but stay put when the new value is already on screen,
// e.g. a range start picked in the leading days of the grid.
watch(anchor, (date) => {
	if (!date) {
		return
	}
	focusedDate.value = date
	if (!cells.value.some(cell => isSameDay(cell.date, date))) {
		view.value = monthOf(date)
	}
})
const monthNames = computed(() => Array.from({length: 12}, (_, month) => formatDate(new Date(2024, month, 1), 'MMMM')))

function setMonth(value: string) {
	view.value = {year: view.value.year, month: Number(value)}
}

function setYear(event: Event) {
	const target = event.target as HTMLInputElement
	const year = Number.parseInt(target.value, 10)
	if (Number.isNaN(year) || year < 1 || year > 9999) {
		target.value = String(view.value.year)
		return
	}
	view.value = {year, month: view.value.month}
}

// Dayjs 'dd' gives the locale's two-letter weekday; a fixed reference week avoids DST oddities.
const weekdayLabels = computed(() => weekdayOrder(weekStart.value)
	.map(day => formatDate(new Date(2024, 0, 7 + day), 'dd')))

function dateKey(date: Date) {
	return formatDate(date, 'YYYY-MM-DD')
}

function isDisabled(date: Date) {
	return props.minDate !== null && startOfDay(date).getTime() < startOfDay(props.minDate).getTime()
}

function cellClasses(cell: CalendarCell) {
	const date = cell.date
	const classes: Record<string, boolean> = {
		'is-outside': !cell.inMonth,
		'is-selected': false,
		'is-range-start': false,
		'is-range-end': false,
		'is-in-range': false,
	}

	if (props.mode !== 'range') {
		classes['is-selected'] = isSameDay(date, props.selected)
		return classes
	}

	const start = props.rangeStart
	// While only the start is picked, preview the span up to the hovered day.
	const end = props.rangeEnd ?? (start && hovered.value && hovered.value > start ? hovered.value : null)

	if (start && isSameDay(date, start)) {
		classes['is-selected'] = true
		classes['is-range-start'] = end !== null && !isSameDay(start, end)
	}
	if (end && isSameDay(date, end)) {
		classes['is-selected'] = true
		classes['is-range-end'] = start !== null && !isSameDay(start, end)
	}
	if (start && end && isDayBetween(date, start, end)) {
		classes['is-in-range'] = true
	}

	return classes
}

function moveMonth(delta: number) {
	const date = new Date(view.value.year, view.value.month + delta, 1)
	view.value = monthOf(date)
}

function pick(date: Date) {
	if (isDisabled(date)) {
		return
	}
	emit('pick', startOfDay(date))
}

async function focusDay(date: Date) {
	focusedDate.value = date
	if (date.getMonth() !== view.value.month || date.getFullYear() !== view.value.year) {
		view.value = monthOf(date)
	}
	await nextTick()
	const el = root.value?.querySelector<HTMLButtonElement>(`.calendar-month__day[data-date="${dateKey(date)}"]`)
	el?.focus()
}

function onKeydown(event: KeyboardEvent) {
	const steps: Record<string, number> = {
		ArrowLeft: -1,
		ArrowRight: 1,
		ArrowUp: -7,
		ArrowDown: 7,
	}
	if (event.key in steps) {
		event.preventDefault()
		focusDay(addDays(focusedDate.value, steps[event.key]))
		return
	}
	if (event.key === 'PageUp' || event.key === 'PageDown') {
		event.preventDefault()
		const date = new Date(focusedDate.value)
		date.setMonth(date.getMonth() + (event.key === 'PageUp' ? -1 : 1))
		focusDay(date)
	}
}
</script>

<style lang="scss" scoped>
.calendar-month {
	--calendar-day-size: 2.125rem;
	// Translucent so the span reads on both light and dark backgrounds (--primary-light is near-white in dark mode).
	--calendar-range-bg: hsla(var(--primary-hsl), .15);
	inline-size: 100%;
	user-select: none;

	&.is-large {
		--calendar-day-size: 2.625rem;
	}
}

.calendar-month__nav {
	display: flex;
	align-items: center;
	justify-content: space-between;
	margin-block-end: .5rem;
}

.calendar-month__nav-button {
	inline-size: 2rem;
	block-size: 2rem;
	display: grid;
	place-items: center;
	border-radius: $radius;
	color: var(--grey-600);
	transition: background-color $transition;

	&:hover {
		background: var(--grey-100);
	}
}

.calendar-month__title {
	display: flex;
	align-items: center;
	gap: .25rem;
}

// Both read as the plain "September 2026" heading; the native controls only show on interaction.
.calendar-month__month-select,
.calendar-month__year-input {
	appearance: none;
	border: 0;
	background: transparent;
	padding: .125rem .25rem;
	border-radius: $radius;
	font-family: $vikunja-font;
	font-weight: 700;
	font-size: 1.05rem;
	color: var(--grey-900);
	cursor: pointer;
	transition: background-color $transition;

	&:hover {
		background: var(--grey-100);
	}

	&:focus-visible {
		outline: none;
		@include focus-ring;
	}
}

.calendar-month__month-select {
	// Shrink to the selected month instead of the widest option (falls back to the full width where unsupported).
	field-sizing: content;
	text-align: end;
}

// type="text" rather than number: Firefox keeps spinner arrows on number inputs regardless of appearance.
.calendar-month__year-input {
	inline-size: 3.25em;
	min-inline-size: 3.25em;
	field-sizing: content;
	color: var(--grey-400);
	font-weight: 500;

	&:focus {
		color: var(--grey-900);
	}
}

.calendar-month__grid {
	display: grid;
	grid-template-columns: repeat(7, 1fr);
}

.calendar-month__weekday {
	text-align: center;
	font-size: .7rem;
	font-weight: 700;
	letter-spacing: .03em;
	text-transform: uppercase;
	color: var(--grey-400);
	padding-block-end: .25rem;
}

.calendar-month__cell {
	display: grid;
	place-items: center;
	block-size: calc(var(--calendar-day-size) + .375rem);

	&.is-in-range {
		background: var(--calendar-range-bg);
	}

	&.is-range-start {
		background: linear-gradient(to right, transparent 50%, var(--calendar-range-bg) 50%);
	}

	&.is-range-end {
		background: linear-gradient(to left, transparent 50%, var(--calendar-range-bg) 50%);
	}
}

.calendar-month__day {
	inline-size: var(--calendar-day-size);
	block-size: var(--calendar-day-size);
	display: grid;
	place-items: center;
	border: 1.5px solid transparent;
	border-radius: $radius;
	background: transparent;
	font-family: inherit;
	font-size: .9rem;
	font-weight: 500;
	font-variant-numeric: tabular-nums;
	color: var(--grey-800);
	cursor: pointer;
	transition: background-color $transition, color $transition;

	.is-large & {
		font-size: 1rem;
	}

	.is-outside & {
		color: var(--grey-300);
	}

	&.is-today {
		border-color: var(--primary);
		color: var(--primary);
		font-weight: 700;
	}

	&:hover:not(:disabled) {
		background: var(--grey-100);
	}

	&:focus-visible {
		outline: none;
		@include focus-ring;
	}

	.is-selected & {
		background: var(--primary);
		border-color: var(--primary);
		color: var(--primary-invert);
		font-weight: 700;
		box-shadow: var(--shadow-sm);

		&:hover {
			background: var(--primary);
		}
	}

	&:disabled {
		color: var(--grey-300);
		cursor: not-allowed;
	}
}
</style>
