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
				<div class="bottom-sheet__body">
					<slot />
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script setup lang="ts">
import {onBeforeUnmount, onMounted} from 'vue'
import {useScrollLock} from '@vueuse/core'

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

// The sheet is another modal layer: the page behind it must not scroll along with it.
const pageScrollLocked = useScrollLock(document.body)
onMounted(() => {
	pageScrollLocked.value = true
})
onBeforeUnmount(() => {
	pageScrollLocked.value = false
})
</script>

<style lang="scss" scoped>
.bottom-sheet {
	position: fixed;
	inset: 0;
	z-index: 1000;
	display: flex;
	flex-direction: column;
	justify-content: flex-end;
}

.bottom-sheet__scrim {
	position: absolute;
	inset: 0;
	background: hsla(0, 0%, 4%, .4);
}

.bottom-sheet__panel {
	position: relative;
	display: flex;
	flex-direction: column;
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
}
</style>
