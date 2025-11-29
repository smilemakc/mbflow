import { describe, it, expect } from 'vitest'
import { generateRandomName } from '@/utils/name-generator'

describe('Name Generator', () => {
    it('generates names in format adjective_noun', () => {
        const name = generateRandomName()

        // Should contain exactly one underscore
        const parts = name.split('_')
        expect(parts).toHaveLength(2)

        // Both parts should be non-empty strings
        expect(parts[0]).toBeTruthy()
        expect(parts[1]).toBeTruthy()

        // Should be lowercase
        expect(name).toBe(name.toLowerCase())
    })

    it('generates different names on multiple calls', () => {
        const names = new Set()

        // Generate 50 names
        for (let i = 0; i < 50; i++) {
            names.add(generateRandomName())
        }

        // Should have high uniqueness (at least 40 unique names out of 50)
        expect(names.size).toBeGreaterThan(40)
    })

    it('generates valid snake_case names', () => {
        for (let i = 0; i < 10; i++) {
            const name = generateRandomName()

            // Should match snake_case pattern
            expect(name).toMatch(/^[a-z]+_[a-z]+$/)
        }
    })

    it('uses consistent format', () => {
        const name = generateRandomName()

        // Should not have leading/trailing underscores
        expect(name).not.toMatch(/^_/)
        expect(name).not.toMatch(/_$/)

        // Should not have multiple consecutive underscores
        expect(name).not.toMatch(/__/)
    })
})
