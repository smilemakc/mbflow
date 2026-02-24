import { useMemo } from 'react';

/**
 * Hook for providing default values to config.
 * Ensures config always has all required fields with defaults.
 *
 * @param config - Config object (may be partial or undefined)
 * @param defaults - Default values for all config fields
 * @returns Complete config with defaults applied
 *
 * @example
 * const safeConfig = useSafeConfig(config, {
 *   operation: 'read',
 *   spreadsheet_id: '',
 *   sheet_name: '',
 * });
 */
export function useSafeConfig<T extends Record<string, any>>(
  config: Partial<T> | undefined,
  defaults: T
): T {
  return useMemo(() => ({
    ...defaults,
    ...config,
  }), [config, defaults]);
}
