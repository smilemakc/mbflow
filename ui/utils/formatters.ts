/**
 * Utility functions for formatting data in the Resources module
 * Single Responsibility: Only handles data formatting
 */

/**
 * Formats bytes into human-readable string
 * @param bytes - Number of bytes
 * @returns Formatted string (e.g., "1.5 MB")
 */
export const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

/**
 * Formats date string into localized format
 * @param dateStr - ISO date string
 * @returns Formatted date string
 */
export const formatDate = (dateStr: string): string => {
  const date = new Date(dateStr);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
};

/**
 * Formats date string into short format (date only)
 * @param dateStr - ISO date string
 * @returns Formatted date string
 */
export const formatShortDate = (dateStr: string): string => {
  const date = new Date(dateStr);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
};

/**
 * Formats currency amount
 * @param amount - Numeric amount
 * @param currency - Currency code (default: USD)
 * @returns Formatted currency string
 */
export const formatCurrency = (amount: number, currency: string = 'USD'): string => {
  return `${amount.toFixed(2)} ${currency}`;
};

/**
 * Generates a unique idempotency key for transactions
 * @param prefix - Optional prefix for the key
 * @returns Unique idempotency key
 */
export const generateIdempotencyKey = (prefix: string = 'tx'): string => {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
};
