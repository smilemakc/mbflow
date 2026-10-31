import '@testing-library/jest-dom/vitest';
import { afterEach, beforeAll } from 'vitest';
import { cleanup } from '@testing-library/react';
import axios, { type AxiosAdapter } from 'axios';
import { apiClient } from '@/lib/api';

// Tests must never reach the network: unmocked requests fail loudly.
const unmocked: AxiosAdapter = async (config) => {
  throw new Error(`Unmocked request: ${(config.method || 'get').toUpperCase()} ${config.baseURL ?? ''}${config.url}`);
};

beforeAll(() => {
  axios.defaults.adapter = unmocked;
  apiClient.defaults.adapter = unmocked;
});

afterEach(() => {
  cleanup();
  try {
    localStorage.clear();
  } catch {
    /* ignore */
  }
});
