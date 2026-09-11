import {calculateDayInterval} from '@/helpers/time/calculateDayInterval'
import {addDays, isSameDay, startOfDay} from '@/helpers/time/calendarGrid'

export const DATE_SHORTCUT_KEYS = ['today', 'tomorrow', 'nextMonday', 'thisWeekend', 'laterThisWeek', 'nextWeek'] as const
export type DateShortcutKey = typeof DATE_SHORTCUT_KEYS[number]

export interface DateShortcut {
	key: DateShortcutKey
	date: Date
}

// Late in the day "today" is pointless, and on a late Sunday "this weekend" would be today as well.
function isPointlessLate(key: DateShortcutKey, now: Date): boolean {
	if (now.getHours() < 21) {
		return false
	}
	return key === 'today' || (key === 'thisWeekend' && now.getDay() === 0)
}

// Several shortcuts can land on the same day (Friday: tomorrow = this weekend); the earlier, more
// basic one wins so no two entries highlight together.
export function buildDateShortcuts(now: Date = new Date()): DateShortcut[] {
	const shortcuts: DateShortcut[] = []
	for (const key of DATE_SHORTCUT_KEYS) {
		if (isPointlessLate(key, now)) {
			continue
		}
		const date = startOfDay(addDays(now, calculateDayInterval(key, now.getDay())))
		if (shortcuts.some(existing => isSameDay(existing.date, date))) {
			continue
		}
		shortcuts.push({key, date})
	}
	return shortcuts
}
