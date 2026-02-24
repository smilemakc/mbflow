import { useState } from 'react';
import { useDagStore } from '@/store/dagStore';

/**
 * Auto-save is DISABLED - workflow saves only before execution.
 * This hook now only provides isSaving state for UI consistency.
 */
export const useAutoSave = () => {
  const { isSaving } = useDagStore();

  // Auto-save disabled - workflow saves only on manual save or before run
  // Keep isSaving for backwards compatibility with Header component
  return { isSaving };
};
