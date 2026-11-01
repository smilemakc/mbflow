import { describe, expect, it, vi } from 'vitest';
import { formatBytes, formatCurrency, formatDate, formatShortDate, generateIdempotencyKey } from './formatters';

describe('formatters', () => {
  it('formats bytes', () => {
    expect(formatBytes(0)).toBe('0 B');
    expect(formatBytes(512)).toBe('512 B');
    expect(formatBytes(1536)).toBe('1.5 KB');
    expect(formatBytes(5 * 1024 * 1024)).toBe('5 MB');
    expect(formatBytes(1024 ** 4)).toBe('1 TB');
  });

  it('formats dates in en-US', () => {
    const iso = new Date(2026, 0, 15, 9, 5).toISOString();
    expect(formatDate(iso)).toMatch(/Jan 15, 2026/);
    expect(formatDate(iso)).toMatch(/09:05/);
    expect(formatShortDate(iso)).toBe('Jan 15, 2026');
  });

  it('formats currency', () => {
    expect(formatCurrency(3)).toBe('3.00 USD');
    expect(formatCurrency(1.005, 'EUR')).toMatch(/^1\.0\d EUR$/);
  });

  it('generates prefixed unique idempotency keys', () => {
    vi.spyOn(Date, 'now').mockReturnValue(1700000000000);
    const a = generateIdempotencyKey();
    const b = generateIdempotencyKey('dep');
    expect(a).toMatch(/^tx-1700000000000-[a-z0-9]+$/);
    expect(b).toMatch(/^dep-1700000000000-/);
    expect(a).not.toBe(generateIdempotencyKey());
  });
});
