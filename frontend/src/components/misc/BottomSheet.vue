<template>
	<Teleport :to="teleportTarget">
		<div
			ref="sheet"
			class="bottom-sheet"
		>
			<div
				class="bottom-sheet__scrim"
				@click.stop="emit('close')"
			/>
			<div
				ref="panel"
				class="bottom-sheet__panel"
				role="dialog"
				aria-modal="true"
				tabindex="-1"
				:aria-label="title || undefined"
			>
				<div class="bottom-sheet__handle" />
				<div
					v-if="title || $slots['header-action']"
					class="bottom-sheet__header"
				>
					<span class="bottom-sheet__title">{{ title }}</span>
					<slot name="header-action" />
				</div>
				<div
					ref="body"
					class="bottom-sheet__body"
				>
					<slot />
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script setup lang="ts">
import {nextTick, onBeforeUnmount, onMounted, ref} from 'vue'
import {onKeyStroke, useEventListener, useScrollLock} from '@vueuse/core'

withDefaults(defineProps<{
	title?: string
}>(), {
	title: '',
})

const emit = defineEmits<{
	close: []
}>()

// Fixed positioning is trapped by ancestor stacking contexts; the open task modal is a top-layer <dialog>, so teleport into it or body would sit beneath.
const openDialogs = document.querySelectorAll('dialog[open]')
const teleportTarget = openDialogs.length > 0 ? openDialogs[openDialogs.length - 1] : document.body

const sheet = ref<HTMLElement | null>(null)
const panel = ref<HTMLElement | null>(null)
const inerted: Element[] = []
const previouslyFocused = document.activeElement as HTMLElement | null

// Nothing behind the sheet may scroll: lock <html>/body (the real page scrollers) and swallow wheel/touch outside the sheet.
const body = ref<HTMLElement | null>(null)
const htmlScrollLocked = useScrollLock(document.documentElement)
const bodyScrollLocked = useScrollLock(document.body)
// Inside a task modal .modal-container is the scroller; lock it too or its scrollbar stays visible and draggable.
const dialogScrollers: {element: HTMLElement, overflow: string}[] = []
onMounted(async () => {
	htmlScrollLocked.value = true
	bodyScrollLocked.value = true

	// aria-modal alone does not stop Tab from reaching the page behind the sheet.
	for (const element of teleportTarget.children) {
		if (element === sheet.value || element.hasAttribute('inert')) {
			continue
		}
		element.setAttribute('inert', '')
		inerted.push(element)
	}

	if (teleportTarget !== document.body) {
		for (const element of teleportTarget.querySelectorAll<HTMLElement>('*')) {
			// The sheet is already teleported in here; locking its own scrollers would trap its content.
			if (sheet.value?.contains(element)) {
				continue
			}
			const overflowY = getComputedStyle(element).overflowY
			if ((overflowY === 'auto' || overflowY === 'scroll') && element.scrollHeight > element.clientHeight) {
				dialogScrollers.push({element, overflow: element.style.overflow})
				element.style.overflow = 'hidden'
			}
		}
	}

	await nextTick()
	if (!panel.value?.contains(document.activeElement)) {
		panel.value?.focus()
	}
})
onBeforeUnmount(() => {
	htmlScrollLocked.value = false
	bodyScrollLocked.value = false
	for (const element of inerted) {
		element.removeAttribute('inert')
	}
	// A consumer restoring focus while the sheet was open (Datepicker on Escape) hit an inert trigger, so focus is nowhere.
	if (previouslyFocused?.isConnected && (document.activeElement === document.body || sheet.value?.contains(document.activeElement))) {
		previouslyFocused.focus()
	}
	for (const {element, overflow} of dialogScrollers) {
		element.style.overflow = overflow
	}
})

// A nested Modal (e.g. datepicker's "Learn how" help) teleports its own <dialog> to body, outside the sheet — it owns the event.
function isFromDialogAboveSheet(event: Event) {
	const dialog = (event.target as Element)?.closest?.('dialog[open]')
	return dialog ? !dialog.contains(sheet.value) : false
}

useEventListener(window, ['wheel', 'touchmove'], (event: Event) => {
	if (isFromDialogAboveSheet(event)) {
		return
	}
	if (!body.value?.contains(event.target as Node)) {
		event.preventDefault()
	}
}, {passive: false, capture: true})

onKeyStroke('Escape', event => {
	// defaultPrevented means an inner control (Multiselect, …) already consumed this Escape.
	if (event.defaultPrevented) {
		return
	}

	if (isFromDialogAboveSheet(event)) {
		return
	}

	// Cancels the close request of a wrapping native <dialog> so only the sheet closes.
	event.preventDefault()
	emit('close')
})
</script>

<style lang="scss" scoped>
.bottom-sheet {
	position: fixed;
	inset-block: 0;
	inset-inline-start: 0;
	// inset: 0 stops short of a classic scrollbar (Firefox with scrollbar-width: thin), leaving a bare strip; 100vw covers it.
	inline-size: 100vw;
	// Above the PWA/update banners (5000), below toasts (9999).
	z-index: 6000;
	display: flex;
	flex-direction: column;
	justify-content: flex-end;
}

.bottom-sheet__scrim {
	position: absolute;
	inset: 0;
	background: hsla(0, 0%, 4%, .4);
	touch-action: none;
}

.bottom-sheet__panel {
	position: relative;
	display: flex;
	flex-direction: column;
	inline-size: 100%;
	overflow-x: hidden;
	max-block-size: 92dvh;
	background: var(--white);
	border-radius: 18px 18px 0 0;
	box-shadow: 0 -8px 30px hsla(var(--grey-500-hsl), .18);
	padding-block-end: env(safe-area-inset-bottom);
}

.bottom-sheet__handle {
	inline-size: 2.5rem;
	block-size: 5px;
	margin: .5rem auto 0;
	border-radius: $radius-rounded;
	background: var(--grey-200);
	flex-shrink: 0;
}

.bottom-sheet__header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: .5rem 1rem .75rem;
	flex-shrink: 0;
}

.bottom-sheet__title {
	font-family: $vikunja-font;
	font-weight: 700;
	font-size: 1.15rem;
	color: var(--grey-900);
}

.bottom-sheet__body {
	min-block-size: 0;
	overflow-y: auto;
	overscroll-behavior: contain;

	// The mobile task modal stretches every descendant .card to viewport height; a card inside the sheet must not be.
	:deep(.card) {
		min-block-size: 0 !important;
	}
}
</style>
