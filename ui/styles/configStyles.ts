/**
 * Centralized style definitions for node configuration components
 *
 * These styles are extracted from existing node configs to ensure consistency
 * across all configuration forms. They follow the existing design patterns
 * found in LLMNodeConfig, TelegramNodeConfig, HTTPNodeConfig, etc.
 */

export const configStyles = {
  // Input fields
  input: 'w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-800 dark:text-slate-200 placeholder-slate-400',

  select: 'w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-800 dark:text-slate-200',

  textarea: 'w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all resize-y text-slate-800 dark:text-slate-200 placeholder-slate-400',

  textareaMonospace: 'w-full px-3 py-2 text-sm font-mono bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all resize-y text-slate-800 dark:text-slate-200 placeholder-slate-400',

  // Labels and hints
  label: 'text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 block',
  labelRequired: 'text-red-500 ml-1',
  hint: 'text-xs text-slate-500 dark:text-slate-400 mt-1 block',

  // Sections
  section: 'space-y-4 rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/50 p-3',
  sectionTitle: 'text-xs font-semibold uppercase text-slate-500 dark:text-slate-400',

  // Checkbox
  checkbox: 'w-4 h-4 rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-2 focus:ring-blue-500/20 transition-colors',
  checkboxLabel: 'text-sm text-slate-700 dark:text-slate-300',

  // Cards
  card: 'bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl',
  cardPadding: 'p-4 md:p-6',
  cardHover: 'hover:shadow-lg transition-shadow',

  // Gradient headers (for node configs)
  gradientHeader: {
    blue: 'bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-900/10 dark:to-indigo-900/10 border border-blue-200 dark:border-blue-800',
    green: 'bg-gradient-to-r from-green-50 to-emerald-50 dark:from-green-900/10 dark:to-emerald-900/10 border border-green-200 dark:border-green-800',
    amber: 'bg-gradient-to-r from-amber-50 to-orange-50 dark:from-amber-900/10 dark:to-orange-900/10 border border-amber-200 dark:border-amber-800',
    orange: 'bg-gradient-to-r from-orange-50 to-amber-50 dark:from-orange-900/10 dark:to-amber-900/10 border border-orange-200 dark:border-orange-800',
    cyan: 'bg-gradient-to-r from-cyan-50 to-blue-50 dark:from-cyan-900/10 dark:to-blue-900/10 border border-cyan-200 dark:border-cyan-800',
    purple: 'bg-gradient-to-r from-purple-50 to-violet-50 dark:from-purple-900/10 dark:to-violet-900/10 border border-purple-200 dark:border-purple-800',
  },

  // Info boxes
  infoBox: {
    info: 'bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4',
    warning: 'bg-amber-50 dark:bg-amber-950/20 border border-amber-200 dark:border-amber-800 rounded-lg p-4',
    success: 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4',
    error: 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4',
  },

  // Auth form styles (gray-based for auth pages)
  authInput: 'w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white transition-colors',
  authLabel: 'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2',
  authError: 'p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg',
} as const;

export type GradientVariant = keyof typeof configStyles.gradientHeader;
export type InfoBoxVariant = keyof typeof configStyles.infoBox;
