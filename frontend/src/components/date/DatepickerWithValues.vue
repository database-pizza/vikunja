<template>
	<div class="datepicker-with-range-container">
		<Popup
			:open="open"
			:ignore-click-classes="ignoreClickClasses"
			sheet-on-mobile
			:sheet-title="$t('input.datepickerRange.date')"
			@update:open="(open) => !open && $emit('update:open', false)"
		>
			<template #content="{isOpen}">
				<div
					class="datepicker-with-range"
					:class="{'is-open': isOpen}"
				>
					<div class="selections">
						<BaseButton
							:class="{'is-active': customRangeActive}"
							@click="setDate(null)"
						>
							{{ $t('misc.custom') }}
						</BaseButton>
						<BaseButton
							v-for="(value, text) in DATE_VALUES"
							:key="text"
							:class="{'is-active': date === value}"
							@click="setDate(value)"
						>
							{{ $t(`input.datepickerRange.values.${text}`) }}
						</BaseButton>
					</div>
					<div class="calendar-container input-group">
						<label class="label">
							{{ $t('input.datepickerRange.date') }}
							<input
								v-model="date"
								class="input"
								type="text"
							>
						</label>

						<CalendarMonth
							:selected="selectedDate"
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

import {parseDateOrString} from '@/helpers/time/parseDateOrString'
import {toISOStringOrNull} from '@/helpers/time/toISOStringOrNull'
import {formatDate} from '@/helpers/time/formatDate'

import Popup from '@/components/misc/Popup.vue'
import CalendarMonth from '@/components/input/datepicker/CalendarMonth.vue'
import {DATE_VALUES} from '@/components/date/dateRanges'
import BaseButton from '@/components/base/BaseButton.vue'
import DatemathHelp from '@/components/date/DatemathHelp.vue'

const props = withDefaults(defineProps<{
	modelValue: string | Date | null,
	open?: boolean
	ignoreClickClasses?: string[]
}>(), {
	open: false,
	ignoreClickClasses: () => [],
})

const emit = defineEmits<{
	'update:modelValue': [value: string | Date | null],
	'update:open': [open: boolean],
}>()

const showHowItWorks = ref(false)

const date = ref<string|Date|null>('')

watch(
	() => props.modelValue,
	newValue => {
		date.value = newValue
	},
)

const selectedDate = computed<Date | null>(() => {
	const raw = date.value instanceof Date ? toISOStringOrNull(date.value) : date.value
	const parsed = parseDateOrString(raw, null)
	return parsed instanceof Date ? parsed : null
})

function emitChanged() {
	emit('update:modelValue', date.value === '' ? null : date.value)
}

watch(() => date.value, emitChanged)

function pickDay(day: Date) {
	date.value = formatDate(day, 'YYYY-MM-DD HH:mm')
}

function setDate(range: string | null) {
	if (range === null) {
		date.value = ''

		return
	}

	date.value = range
}

const customRangeActive = computed<boolean>(() => {
	return !Object.values(DATE_VALUES).some(d => date.value === d)
})
</script>

<style lang="scss" scoped>
.datepicker-with-range-container {
	position: relative;
}

:deep(.popup) {
	z-index: 10;
	margin-block-start: 1rem;
	border-radius: $radius;
	border: 1px solid var(--grey-200);
	background-color: var(--white);
	box-shadow: $shadow;

	&.is-open {
		inline-size: 560px;
		max-inline-size: calc(100vw - 2rem);
	}

	&.is-sheet {
		inline-size: auto;
		margin: 0;
		border: 0;
		box-shadow: none;
		background: transparent;
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

	> .label {
		margin-block-end: .5rem;
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
