import { useState, useEffect } from 'react';

/**
 * Generic hook for node config state management.
 * Replaces the pattern of local state + useEffect + onChange propagation.
 *
 * @param initialConfig - Initial config object from node
 * @param onChange - Callback to propagate changes to parent
 * @returns Tuple of [config, updateConfig function]
 *
 * @example
 * const [config, updateConfig] = useNodeConfig(nodeConfig, onChange);
 * updateConfig({ model: 'gpt-4' }); // Merges and propagates
 */
export function useNodeConfig<T extends Record<string, any>>(
  initialConfig: T,
  onChange: (config: T) => void
): [T, (updates: Partial<T>) => void] {
  const [localConfig, setLocalConfig] = useState<T>({ ...initialConfig });

  useEffect(() => {
    if (JSON.stringify(initialConfig) !== JSON.stringify(localConfig)) {
      setLocalConfig({ ...initialConfig });
    }
  }, [initialConfig]);

  const updateConfig = (updates: Partial<T>) => {
    const newConfig = { ...localConfig, ...updates };
    setLocalConfig(newConfig);
    onChange(newConfig);
  };

  return [localConfig, updateConfig];
}
