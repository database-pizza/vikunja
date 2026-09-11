import {describe, expect, it} from 'vitest'
import {buildDateShortcuts} from './dateShortcuts'

const keys = (date: Date) => buildDateShortcuts(date).map(s => s.key)

describe('buildDateShortcuts', () => {
	it('drops shortcuts that land on the same day as an earlier one', () => {
		// Friday: tomorrow is the weekend, "later this week" is today
		expect(keys(new Date(2026, 8, 11, 10))).toEqual(['today', 'tomorrow', 'nextMonday', 'nextWeek'])
		// Monday: "next Monday" resolves to today
		expect(keys(new Date(2026, 8, 14, 10))).toEqual(['today', 'tomorrow', 'thisWeekend', 'laterThisWeek', 'nextWeek'])
	})

	it('keeps distinct days on a midweek day', () => {
		// Tuesday
		expect(keys(new Date(2026, 8, 15, 10))).toEqual(['today', 'tomorrow', 'nextMonday', 'thisWeekend', 'laterThisWeek', 'nextWeek'])
	})

	it('hides today late in the evening', () => {
		expect(keys(new Date(2026, 8, 15, 22))).not.toContain('today')
	})

	it('resolves each shortcut to the start of its day', () => {
		const shortcuts = buildDateShortcuts(new Date(2026, 8, 15, 10, 30))
		for (const {date} of shortcuts) {
			expect(date.getHours()).toBe(0)
		}
	})
})
