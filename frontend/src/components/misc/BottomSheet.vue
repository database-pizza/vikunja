<template>
	<Teleport :to="teleportTarget">
		<div class="bottom-sheet">
			<div
				class="bottom-sheet__scrim"
				@click.stop="emit('close')"
			/>
			<div
				class="bottom-sheet__panel"
				role="dialog"
				aria-modal="true"
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
import {onBeforeUnmount, onMounted, ref} from 'vue'
import {useEventListener, useScrollLock} from '@vueuse/core'

withDefaults(defineProps<{
	title?: string
}>(), {
	title: '',
})

const emit = defineEmits<{
	close: []
}>()

// Fixed positioning is trapped by ancestor stacking contexts (the app header would cover the sheet), so render at
// the top of the tree. A task modal is a top-layer <dialog>, and body would sit beneath it, so go there instead.
const openDialogs = document.querySelectorAll('dialog[open]')
const teleportTarget = openDialogs.length > 0 ? openDialogs[openDialogs.length - 1] : document.body

// The sheet is another modal layer: nothing behind it may scroll. <html> is the page scroller here (body's
// overflow does not propagate to the viewport because html sets its own), and inside a task modal the scroller
// is a descendant of the dialog, so on top of the overflow locks every wheel/touch outside the sheet is swallowed.
const body = ref<HTMLElement | null>(null)
const htmlScrollLocked = useScrollLock(document.documentElement)
const bodyScrollLocked = useScrollLock(document.body)
// Inside a task modal the page does not scroll at all; .modal-container does, and its classic scrollbar
// would stay visible (and draggable) next to the sheet unless it is locked as well.
const dialogScrollers: {element: HTMLElement, overflow: string}[] = []
onMounted(() => {
	htmlScrollLocked.value = true
	bodyScrollLocked.value = true
	if (teleportTarget === document.body) {
		return
	}
	for (const element of teleportTarget.querySelectorAll<HTMLElement>('*')) {
		const overflowY = getComputedStyle(element).overflowY
		if ((overflowY === 'auto' || overflowY === 'scroll') && element.scrollHeight > element.clientHeight) {
			dialogScrollers.push({element, overflow: element.style.overflow})
			element.style.overflow = 'hidden'
		}
	}
})
onBeforeUnmount(() => {
	htmlScrollLocked.value = false
	bodyScrollLocked.value = false
	for (const {element, overflow} of dialogScrollers) {
		element.style.overflow = overflow
	}
})
useEventListener(window, ['wheel', 'touchmove'], (event: Event) => {
	if (!body.value?.contains(event.target as Node)) {
		event.preventDefault()
	}
}, {passive: false, capture: true})
</script>

<style lang="scss" scoped>
.bottom-sheet {
	position: fixed;
	inset-block: 0;
	inset-inline-start: 0;
	// inset: 0 stops short of a classic scrollbar (Firefox with scrollbar-width: thin), leaving a bare strip; 100vw covers it.
	inline-size: 100vw;
	z-index: 1000;
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
