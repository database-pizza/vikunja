<template>
	<div class="datepicker">
		<SimpleButton
			ref="triggerButton"
			class="show"
			:disabled="disabled || undefined"
			@click.stop="toggleDatePopup"
		>
			<i v-if="date === null && emptyLabel !== ''">{{ emptyLabel }}</i>
			<template v-else>
				{{ date === null ? chooseDateLabel : formatDisplayDate(date) }}
			</template>
		</SimpleButton>

		<!-- No Teleport: the task modal is a top-layer <dialog>, anything moved to body would sit beneath it. -->
		<CustomTransition name="fade">
			<component
				:is="isMobile ? BottomSheet : 'div'"
				v-if="show"
				v-bind="isMobile ? {title: title, onClose: close} : {class: 'datepicker-layer'}"
			>
				<template
					v-if="isMobile"
					#header-action
				>
					<BaseButton
						class="datepicker-popup__clear"
						@click.stop="clear"
					>
						{{ $t('input.datepicker.clear') }}
					</BaseButton>
				</template>
				<div
					ref="datepickerPopup"
					class="datepicker-popup"
					:class="`datepicker-popup--${layout}`"
					:role="isMobile ? undefined : 'dialog'"
					:aria-label="isMobile ? undefined : title"
					tabindex="-1"
					@keydown.esc.stop="closeViaEsc"
				>
					<DatepickerInline
						v-model="date"
						:show-shortcuts="showShortcuts"
						:shortcuts-layout="isMobile ? 'top' : layout"
						:large="isMobile"
						:min-date="minDate"
						@update:modelValue="updateData"
					/>

					<div class="datepicker-popup__footer">
						<div class="datepicker-popup__summary">
							<Icon
								:icon="['far', 'calendar-check']"
								class="datepicker-popup__summary-icon"
							/>
							<span>{{ date === null ? $t('input.datepicker.noDate') : formatDate(date, summaryFormat) }}</span>
						</div>
						<BaseButton
							v-if="isDev && !isMobile"
							v-tooltip="'Dev: toggle shortcut layout'"
							class="datepicker-popup__layout-toggle"
							@click.stop="toggleLayout"
						>
							{{ layout === 'sidebar' ? 'A' : 'B' }}
						</BaseButton>
						<XButton
							v-if="!isMobile"
							variant="secondary"
							:shadow="false"
							@click.stop="clear"
						>
							{{ $t('input.datepicker.clear') }}
						</XButton>
						<XButton
							v-cy="'closeDatepicker'"
							class="datepicker__close-button"
							:shadow="false"
							@click.stop="close"
						>
							{{ $t('misc.confirm') }}
						</XButton>
					</div>
				</div>
			</component>
		</CustomTransition>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, onMounted, onBeforeUnmount, toRef, watch, nextTick} from 'vue'
import {useI18n} from 'vue-i18n'

import BaseButton from '@/components/base/BaseButton.vue'
import BottomSheet from '@/components/misc/BottomSheet.vue'
import CustomTransition from '@/components/misc/CustomTransition.vue'
import DatepickerInline from '@/components/input/DatepickerInline.vue'
import SimpleButton from '@/components/input/SimpleButton.vue'

import {formatDate, formatDisplayDate} from '@/helpers/time/formatDate'
import {closeWhenClickedOutside} from '@/helpers/closeWhenClickedOutside'
import {createDateFromString} from '@/helpers/time/createDateFromString'
import {useDatepickerLayout} from '@/composables/useDatepickerLayout'
import {useTimeFormat} from '@/composables/useTimeFormat'
import {TIME_FORMAT} from '@/constants/timeFormat'
import {useIsMobile} from '@/composables/useIsMobile'

const props = withDefaults(defineProps<{
	modelValue: Date | null | string,
	// Trigger text while no date is set.
	chooseDateLabel?: string,
	// Dialog title, e.g. the attribute name ("Due Date").
	title?: string,
	disabled?: boolean,
	showShortcuts?: boolean,
	// When the value is null, show this (italic) instead of chooseDateLabel.
	emptyLabel?: string,
	minDate?: Date | null,
}>(), {
	chooseDateLabel: () => {
		const {t} = useI18n({useScope: 'global'})
		return t('input.datepicker.chooseDate')
	},
	title: () => {
		const {t} = useI18n({useScope: 'global'})
		return t('input.datepicker.chooseDate')
	},
	disabled: false,
	showShortcuts: true,
	emptyLabel: '',
	minDate: null,
})

const emit = defineEmits<{
	'update:modelValue': [value: Date | null],
	'close': [value: boolean],
	'closeOnChange': [value: boolean],
}>()

const isDev = import.meta.env.DEV
const layout = useDatepickerLayout()
const isMobile = useIsMobile()

const {store: timeFormat} = useTimeFormat()
const summaryFormat = computed(() => timeFormat.value === TIME_FORMAT.HOURS_24 ? 'ddd, ll HH:mm' : 'ddd, ll hh:mm A')

const date = ref<Date | null>(null)
const show = ref(false)
const changed = ref(false)

onMounted(() => document.addEventListener('click', hideDatePopup))
onBeforeUnmount(() => document.removeEventListener('click', hideDatePopup))

const modelValue = toRef(props, 'modelValue')
watch(
	modelValue,
	setDateValue,
	{immediate: true},
)

function setDateValue(dateString: string | Date | null) {
	if (dateString === null) {
		date.value = null
		return
	}
	date.value = createDateFromString(dateString)
}

function updateData() {
	changed.value = true
	emit('update:modelValue', date.value ?? null)
}

function clear() {
	date.value = null
	updateData()
}

function toggleLayout() {
	layout.value = layout.value === 'sidebar' ? 'top' : 'sidebar'
}

function toggleDatePopup() {
	if (props.disabled) {
		return
	}

	show.value = !show.value
}

const datepickerPopup = ref<HTMLElement | null>(null)
const triggerButton = ref<InstanceType<typeof SimpleButton> | null>(null)

// nextTick: the popup only exists once the v-if transition renders it.
watch(show, async (isOpen) => {
	if (!isOpen) {
		return
	}
	await nextTick()
	datepickerPopup.value?.focus()
})

function hideDatePopup(e: MouseEvent) {
	if (show.value && datepickerPopup.value) {
		closeWhenClickedOutside(e, datepickerPopup.value, close)
	}
}

function close() {
	// Kind of dirty, but the timeout allows us to enter a time and click on "confirm" without
	// having to click on another input field before it is actually used.
	setTimeout(() => {
		show.value = false
		emit('close', changed.value)
		if (changed.value) {
			changed.value = false
			emit('closeOnChange', changed.value)
		}
	}, 200)
}

function closeViaEsc() {
	// close() defers unmount by 200ms; focus would otherwise drop to <body>.
	triggerButton.value?.focus()
	close()
}
</script>

<style lang="scss" scoped>
.datepicker-layer {
	position: absolute;
	z-index: 99;
}

.datepicker-popup {
	position: relative;
	background: var(--white);
	border: 1px solid var(--grey-200);
	border-radius: $radius-large;
	box-shadow: var(--shadow-md);
	overflow: hidden;
	inline-size: 400px;

	&--sidebar {
		inline-size: 548px;
	}

	.bottom-sheet & {
		inline-size: 100%;
		border: 0;
		border-radius: 0;
		box-shadow: none;
	}
}

.datepicker-popup__clear {
	color: var(--primary);
	font-family: $family-sans-serif;
	font-weight: 700;
	font-size: .85rem;
	text-transform: uppercase;
	letter-spacing: .04em;
}

.datepicker-popup__footer {
	display: flex;
	align-items: center;
	gap: .75rem;
	padding: .75rem 1rem;
	border-block-start: 1px solid var(--grey-200);
	background: var(--grey-50);

	.bottom-sheet & {
		flex-direction: column;
		align-items: stretch;
		gap: .5rem;
		padding-block-end: 1rem;
		background: var(--white);
	}
}

.datepicker-popup__summary {
	display: flex;
	align-items: center;
	gap: .5rem;
	flex: 1;
	min-inline-size: 0;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
	font-size: .85rem;
	font-weight: 600;
	color: var(--grey-700);
	font-variant-numeric: tabular-nums;
}

.datepicker-popup__summary-icon {
	color: var(--primary);
}

.datepicker-popup__layout-toggle {
	padding: .125rem .5rem;
	border: 1px dashed var(--grey-300);
	border-radius: $radius;
	font-size: .75rem;
	color: var(--grey-500);
}

.datepicker__close-button {
	.bottom-sheet & {
		min-block-size: 44px;
	}
}
</style>
