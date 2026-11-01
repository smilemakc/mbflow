import { describe, expect, it, vi } from 'vitest';
import { getAccessToken, getRefreshToken, REFRESH_TOKEN_KEY, TOKEN_KEY } from './authStorage';

describe('authStorage', () => {
  it('reads tokens from localStorage', () => {
    expect(getAccessToken()).toBeNull();
    expect(getRefreshToken()).toBeNull();
    localStorage.setItem(TOKEN_KEY, 'a');
    localStorage.setItem(REFRESH_TOKEN_KEY, 'r');
    expect(getAccessToken()).toBe('a');
    expect(getRefreshToken()).toBe('r');
  });

  it('returns null when storage is unavailable', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked');
    });
    expect(getAccessToken()).toBeNull();
    expect(getRefreshToken()).toBeNull();
  });
});
