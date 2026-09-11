<template>
	<slot
		name="trigger"
		:is-open="openValue"
		:toggle="toggle"
		:close="close"
	/>
	<div
		ref="popup"
		class="popup"
		:class="{
			'is-open': openValue,
			'has-overflow': hasOverflow && openValue,
			'is-sheet': asSheet,
		}"
		:style="floatingStyle"
		:inert="!openValue"
		@focusin="rememberFocusEntered"
	>
		<!-- Closed sheet renders nothing: the popup box is static in sheet mode and would reserve the content's height. -->
		<template v-if="asSheet">
			<BottomSheet
				v-if="openValue"
				:title="sheetTitle"
				@close="close"
			>
				<slot
					name="content"
					:is-open="openValue"
					:toggle="toggle"
					:close="close"
				/>
			</BottomSheet>
		</template>
		<slot
			v-else
			name="content"
			:is-open="openValue"
			:toggle="toggle"
			:close="close"
		/>
	</div>
</template>

<script setup lang="ts">
import {computed, onScopeDispose, ref, watch, watchEffect} from 'vue'
import {onClickOutside, onKeyStroke} from '@vueuse/core'
import {autoUpdate, computePosition, flip, offset, shift, type Placement} from '@floating-ui/dom'

import BottomSheet from '@/components/misc/BottomSheet.vue'
import {useIsMobile} from '@/composables/useIsMobile'

const props = withDefaults(defineProps<{
	hasOverflow?: boolean
	open?: boolean
	ignoreClickClasses?: string[]
	// Anchors the popup to the trigger with floating-ui (flips and shifts to stay on screen). Without it, consumers position via CSS.
	placement?: Placement
	// Below the tablet breakpoint the content opens as a bottom sheet instead.
	sheetOnMobile?: boolean
	sheetTitle?: string
}>(), {
	hasOverflow: false,
	open: false,
	ignoreClickClasses: () => [],
	placement: undefined,
	sheetOnMobile: false,
	sheetTitle: '',
})

const emit = defineEmits<{
	'update:open': [open: boolean]
}>()

defineSlots<{
	trigger(props: {
		isOpen: boolean,
		toggle: () => boolean,
		close: () => void,
	}) : void
	content(props: {
		isOpen: boolean,
		toggle: () => boolean,
		close: () => void
	}): void
}>()

// eslint-disable-next-line vue/no-setup-props-reactivity-loss
const openValue = ref(props.open)
watchEffect(() => {
	openValue.value = props.open
})

function close() {
	if (!openValue.value) {
		return
	}
	openValue.value = false
	emit('update:open', false)
}

// onClickOutside listens in the capture phase, so a trigger's `@click.stop` cannot keep it from
// closing first — without this guard the trigger's own toggle() reopens what the same click closed.
let closedByClickOutside = false

function toggle() {
	if (closedByClickOutside) {
		closedByClickOutside = false
		return false
	}
	openValue.value = !openValue.value
	emit('update:open', openValue.value)
	return openValue.value
}

const popup = ref<HTMLElement | null>(null)

const isMobile = useIsMobile()
const asSheet = computed(() => props.sheetOnMobile && isMobile.value)

const floatingStyle = ref<Record<string, string>>({})

async function updatePosition() {
	// The trigger slot renders right before the popup, so its last root is the anchor.
	const reference = popup.value?.previousElementSibling
	if (!reference || !popup.value || !props.placement) {
		return
	}
	const {x, y} = await computePosition(reference, popup.value, {
		placement: props.placement,
		strategy: 'absolute',
		middleware: [offset(4), flip({padding: 8}), shift({padding: 8})],
	})
	floatingStyle.value = {left: `${x}px`, top: `${y}px`}
}

let stopAutoUpdate: (() => void) | null = null
watch([openValue, asSheet], ([open, sheet]) => {
	stopAutoUpdate?.()
	stopAutoUpdate = null
	if (!open || sheet || !props.placement || !popup.value?.previousElementSibling) {
		floatingStyle.value = {}
		return
	}
	stopAutoUpdate = autoUpdate(popup.value.previousElementSibling, popup.value, updatePosition)
}, {flush: 'post'})

onScopeDispose(() => {
	stopAutoUpdate?.()
	stopAutoUpdate = null
})

let lastFocused: HTMLElement | null = null
let focusEnteredPopup = false

function rememberFocusEntered() {
	focusEnteredPopup = true
}

// Pre-flush so focus is restored before `inert` blurs it to <body>; immediate so a popup mounted open still records its trigger.
watch(openValue, open => {
	if (open) {
		lastFocused = document.activeElement as HTMLElement | null
		return
	}

	// An outside click blurs to <body> at mousedown, before onClickOutside fires, so <body> still means nothing else took focus.
	const active = document.activeElement
	if (focusEnteredPopup && lastFocused?.isConnected && (popup.value?.contains(active) || active === document.body)) {
		lastFocused.focus()
	}
	lastFocused = null
	focusEnteredPopup = false
}, {immediate: true})

onClickOutside(popup, (event) => {
	const target = event.target as HTMLElement
	// The mobile sheet is teleported out of the popup element, so it must not count as "outside".
	// Checked here rather than via the ignore option: that relies on a bubbling pointerdown, which
	// content like DeferTask stops.
	if (target?.closest?.('.bottom-sheet')) {
		return
	}
	// Check if the click target has any of the ignored classes
	if (target?.classList && props.ignoreClickClasses.some(className => target.classList.contains(className))) {
		return
	}
	if (!openValue.value) {
		return
	}
	closedByClickOutside = true
	setTimeout(() => {
		closedByClickOutside = false
	})
	close()
})

onKeyStroke('Escape', event => {
	// defaultPrevented means an inner control (Multiselect, …) already consumed this Escape.
	if (!openValue.value || event.defaultPrevented) {
		return
	}

	// Scope to the popup owning focus — lastFocused is the trigger, which keeps focus after opening.
	const target = event.target as Node | null
	const insideSheet = target instanceof Element && target.closest('.bottom-sheet') !== null
	if (!target || (!popup.value?.contains(target) && !insideSheet && target !== lastFocused)) {
		return
	}

	// Cancels the close request of a wrapping native <dialog> so only the popup closes.
	event.preventDefault()
	close()
})
</script>

<style scoped lang="scss">
.popup {
	transition: opacity $transition;
	opacity: 0;
	visibility: hidden;
	block-size: 0;
	overflow: hidden;
	position: absolute;
	inset-block-start: 1rem;
	z-index: 100;

	&.is-open {
		opacity: 1;
		visibility: visible;
		block-size: auto;
	}

	// The sheet inside positions itself against the viewport; the popup box must not clip or offset it.
	&.is-sheet {
		position: static;
		block-size: auto;
		overflow: visible;
	}
}
</style>
