import {describe, expect, it} from 'vitest'
import {Schema} from '@tiptap/pm/model'

import {decorateDocument} from './highlighter'

const schema = new Schema({
	nodes: {
		doc: {content: 'paragraph+'},
		paragraph: {content: 'text*'},
		text: {inline: true},
	},
})

function filterDocument(value: string) {
	return schema.node('doc', null, [
		schema.node('paragraph', null, [schema.text(value)]),
	])
}

describe('filter highlighter', () => {
	it('uses loaded label data when recalculating decorations', () => {
		const doc = filterDocument('labels = "Work"')

		const before = decorateDocument(doc, [])
		const after = decorateDocument(doc, [{id: 1, title: 'Work', hex_color: 'ff006e'}])

		expect(before.find()).not.toEqual(after.find())
	})

	it('marks unquoted date values as clickable', () => {
		const decorations = decorateDocument(filterDocument('dueDate < now/w+1w'), []).find()
		const dateValue = decorations.find(d => d.type?.attrs?.class === 'date-value')
		expect(dateValue).toBeDefined()
		expect(dateValue?.type?.attrs?.['data-date-value']).toBe('now/w+1w')
	})

	it('marks unquoted label values', () => {
		const decorations = decorateDocument(filterDocument('labels = Work'), [{id: 1, title: 'Work', hex_color: 'ff006e'}]).find()
		expect(decorations.some(d => d.type?.attrs?.class === 'label-value')).toBe(true)
	})
})
