import { describe, it, expect } from 'vitest'
import { toSnakeCase, toTitleCase } from '@/utils/formatting'

describe('Formatting Utils', () => {
    describe('toSnakeCase', () => {
        it('converts simple text to snake_case', () => {
            expect(toSnakeCase('Hello World')).toBe('hello_world')
            expect(toSnakeCase('My Node Name')).toBe('my_node_name')
        })

        it('handles already snake_case text', () => {
            expect(toSnakeCase('already_snake_case')).toBe('already_snake_case')
        })

        it('removes special characters', () => {
            expect(toSnakeCase('Hello@World!')).toBe('hello_world')
            expect(toSnakeCase('Test-Node-123')).toBe('test_node_123')
        })

        it('handles multiple spaces and underscores', () => {
            expect(toSnakeCase('  multiple   spaces  ')).toBe('multiple_spaces')
            expect(toSnakeCase('___multiple___underscores___')).toBe('multiple_underscores')
        })

        it('preserves numbers', () => {
            expect(toSnakeCase('Node 123')).toBe('node_123')
            expect(toSnakeCase('transform_1234567890')).toBe('transform_1234567890')
        })

        it('handles empty string', () => {
            expect(toSnakeCase('')).toBe('')
        })
    })

    describe('toTitleCase', () => {
        it('converts snake_case to Title Case', () => {
            expect(toTitleCase('hello_world')).toBe('Hello World')
            expect(toTitleCase('my_node_name')).toBe('My Node Name')
        })

        it('handles single word', () => {
            expect(toTitleCase('transform')).toBe('Transform')
            expect(toTitleCase('http')).toBe('Http')
        })

        it('handles already Title Case (with underscores)', () => {
            expect(toTitleCase('Already_Title_Case')).toBe('Already Title Case')
        })

        it('handles empty string', () => {
            expect(toTitleCase('')).toBe('')
        })

        it('capitalizes each word correctly', () => {
            expect(toTitleCase('transform_brave_owl')).toBe('Transform Brave Owl')
            expect(toTitleCase('http_swift_panda')).toBe('Http Swift Panda')
        })
    })

    describe('Round-trip conversion', () => {
        it('maintains consistency through conversions', () => {
            const original = 'My Test Node'
            const snake = toSnakeCase(original)
            const title = toTitleCase(snake)

            expect(snake).toBe('my_test_node')
            expect(title).toBe('My Test Node')
        })
    })
})
