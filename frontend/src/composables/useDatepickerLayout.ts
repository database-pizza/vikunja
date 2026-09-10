import {useLocalStorage} from '@vueuse/core'

export type DatepickerLayout = 'sidebar' | 'top'

// Temporary: lets us compare both shortcut layouts in the running app before settling on one.
export function useDatepickerLayout() {
	return useLocalStorage<DatepickerLayout>('vikunja.datepickerLayout', 'sidebar')
}
