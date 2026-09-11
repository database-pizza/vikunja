import {computed, ref, toValue, type MaybeRefOrGetter} from 'vue'

export function useRangePick(
	start: MaybeRefOrGetter<Date | null>,
	end: MaybeRefOrGetter<Date | null>,
	onPicked: (start: Date, end: Date) => void,
) {
	// Only the first click of a new range; the model is untouched until the end is picked too.
	const pendingStart = ref<Date | null>(null)

	const rangeStart = computed(() => pendingStart.value ?? toValue(start))
	const rangeEnd = computed(() => pendingStart.value ? null : toValue(end))

	function reset() {
		pendingStart.value = null
	}

	function pickDay(day: Date) {
		if (pendingStart.value === null) {
			pendingStart.value = day
			return
		}

		const [pickedStart, pickedEnd] = day < pendingStart.value ? [day, pendingStart.value] : [pendingStart.value, day]
		reset()
		onPicked(pickedStart, pickedEnd)
	}

	return {pendingStart, rangeStart, rangeEnd, pickDay, reset}
}
