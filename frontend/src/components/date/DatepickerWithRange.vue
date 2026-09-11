<template>
	<div class="datepicker-with-range-container">
		<Popup
			placement="bottom-start"
			sheet-on-mobile
			:sheet-title="$t('misc.dateRange')"
		>
			<template #trigger="{toggle}">
				<slot
					name="trigger"
					:toggle="toggle"
					:button-text="buttonText"
				/>
			</template>
			<template #content="{isOpen}">
				<div
					class="datepicker-with-range"
					:class="{'is-open': isOpen}"
				>
					<div class="selections">
						<BaseButton
							:class="{'is-active': customRangeActive}"
							@click="setDateRange(null)"
						>
							{{ $t('misc.custom') }}
						</BaseButton>
						<BaseButton
							v-for="(value, text) in DATE_RANGES"
							:key="text"
							:class="{'is-active': from === value[0] && to === value[1]}"
							@click="setDateRange([...value])"
						>
							{{ $t(`input.datepickerRange.ranges.${text}`) }}
						</BaseButton>
					</div>
					<div class="calendar-container input-group">
						<div class="date-fields">
							<label class="label">
								{{ $t('input.datepickerRange.from') }}
								<input
									v-model="from"
									class="input"
									type="text"
								>
							</label>
							<label class="label">
								{{ $t('input.datepickerRange.to') }}
								<input
									v-model="to"
									class="input"
									type="text"
								>
							</label>
						</div>

						<CalendarMonth
							mode="range"
							:range-start="rangeStart"
							:range-end="rangeEnd"
							@pick="pickDay"
						/>

						<p>
							{{ $t('input.datemathHelp.canuse') }}
						</p>

						<BaseButton
							class="has-text-primary"
							@click="showHowItWorks = true"
						>
							{{ $t('input.datemathHelp.learnhow') }}
						</BaseButton>

						<Modal
							:enabled="showHowItWorks"
							:overflow="true"
							variant="hint-modal"
							@close="() => showHowItWorks = false"
						>
							<DatemathHelp />
						</Modal>
					</div>
				</div>
			</template>
		</Popup>
	</div>
</template>

<script lang="ts" setup>
import {computed, ref, watch} from 'vue'
import {useI18n} from 'vue-i18n'

import {parseDateOrString} from '@/helpers/time/parseDateOrString'
import {formatDate} from '@/helpers/time/formatDate'

import Popup from '@/components/misc/Popup.vue'
import CalendarMonth from '@/components/input/datepicker/CalendarMonth.vue'
import {DATE_RANGES} from '@/components/date/dateRanges'
import BaseButton from '@/components/base/BaseButton.vue'
import DatemathHelp from '@/components/date/DatemathHelp.vue'

const props = defineProps<{
	// null for a side that's been cleared (the Custom option) — emitted, so accepted too.
	modelValue: {
		dateFrom: Date | string | null,
		dateTo: Date | string | null,
	},
}>()

const emit = defineEmits<{
	'update:modelValue': [value: {
		dateFrom: Date | string | null,
		dateTo: Date | string | null
	}]
}>()

const {t} = useI18n({useScope: 'global'})

const showHowItWorks = ref(false)

const from = ref('')
const to = ref('')

watch(
	() => props.modelValue,
	newValue => {
		from.value = typeof newValue.dateFrom === 'string' ? newValue.dateFrom : (newValue.dateFrom?.toISOString() ?? '')
		to.value = typeof newValue.dateTo === 'string' ? newValue.dateTo : (newValue.dateTo?.toISOString() ?? '')
	},
)

function emitChanged() {
	const args = {
		dateFrom: from.value === '' ? null : from.value,
		dateTo: to.value === '' ? null : to.value,
	}
	emit('update:modelValue', args)
}

watch(() => from.value, emitChanged)
watch(() => to.value, emitChanged)

function asDate(value: string): Date | null {
	const parsed = parseDateOrString(value, null)
	return parsed instanceof Date ? parsed : null
}

// Only the first click of a new range; the model is untouched until the end is picked too.
const pendingStart = ref<Date | null>(null)

const rangeStart = computed(() => pendingStart.value ?? asDate(from.value))
const rangeEnd = computed(() => pendingStart.value ? null : asDate(to.value))

function pickDay(day: Date) {
	if (pendingStart.value === null) {
		pendingStart.value = day
		return
	}

	const [start, end] = day < pendingStart.value ? [day, pendingStart.value] : [pendingStart.value, day]
	pendingStart.value = null
	// Inclusive end: a range ending "Sep 22" should cover the whole 22nd.
	from.value = formatDate(start, 'YYYY-MM-DD 00:00')
	to.value = formatDate(end, 'YYYY-MM-DD 23:59')
}

function setDateRange(range: string[] | null) {
	pendingStart.value = null
	if (range === null) {
		from.value = ''
		to.value = ''

		return
	}

	from.value = range[0] ?? ''
	to.value = range[1] ?? ''
}

const customRangeActive = computed<boolean>(() => {
	return !Object.values(DATE_RANGES).some(range => from.value === range[0] && to.value === range[1])
})

const buttonText = computed<string>(() => {
	if (from.value === '' || to.value === '') {
		return t('task.show.select')
	}

	// Show the preset's name when the range matches one, rather than the raw datemath.
	const preset = Object.entries(DATE_RANGES).find(
		([, range]) => from.value === range[0] && to.value === range[1],
	)
	if (preset) {
		return t(`input.datepickerRange.ranges.${preset[0]}`)
	}

	return t('input.datepickerRange.fromto', {
		from: from.value,
		to: to.value,
	})
})
</script>

<style lang="scss" scoped>
.datepicker-with-range-container {
	position: relative;
}

:deep(.popup) {
	z-index: 10;
	border-radius: $radius;
	border: 1px solid var(--grey-200);
	background-color: var(--white);
	box-shadow: $shadow;

	&.is-open {
		inline-size: 560px;
		max-inline-size: calc(100vw - 2rem);
	}
}

.datepicker-with-range {
	display: flex;
	inline-size: 100%;
}

.calendar-container {
	inline-size: 70%;
	border-inline-start: 1px solid var(--grey-200);
	padding: 1rem;
	font-size: .9rem;

	.label, .input {
		font-size: .9rem;
	}
}

.date-fields {
	display: flex;
	gap: .5rem;
	margin-block-end: .5rem;

	.label {
		flex: 1;
	}
}

.selections {
	inline-size: 30%;
	display: flex;
	flex-direction: column;
	padding-block-start: .5rem;
	overflow-y: auto;
	max-block-size: 30rem;

	button {
		display: block;
		inline-size: 100%;
		text-align: start;
		padding: .5rem 1rem;
		transition: $transition;
		font-size: .9rem;
		color: var(--text);
		background: transparent;
		border: 0;
		cursor: pointer;

		&.is-active {
			color: var(--primary);
		}

		&:hover, &.is-active {
			background-color: var(--grey-100);
		}
	}
}

.bottom-sheet .datepicker-with-range {
	flex-direction: column;
}

.bottom-sheet .selections {
	flex-direction: row;
	flex-wrap: nowrap;
	inline-size: 100%;
	max-block-size: none;
	gap: .5rem;
	padding: .75rem 1rem;
	overflow-x: auto;
	overflow-y: hidden;
	border-block-end: 1px solid var(--grey-200);
	scrollbar-width: none;

	&::-webkit-scrollbar {
		display: none;
	}

	button {
		inline-size: auto;
		padding: .3125rem .6875rem;
		border: 1px solid var(--grey-200);
		border-radius: $radius-rounded;
		white-space: nowrap;
		font-weight: 600;

		&.is-active {
			background: var(--primary);
			border-color: var(--primary);
			color: var(--primary-invert);
		}
	}
}

.bottom-sheet .calendar-container {
	inline-size: 100%;
	border-inline-start: 0;
}
</style>
