/**
 * Formatting utilities
 */

/**
 * Converts a string to snake_case.
 * Replaces spaces and non-alphanumeric characters with underscores.
 * Converts to lowercase.
 * Example: "My Node Name" -> "my_node_name"
 */
export function toSnakeCase(str: string): string {
    return str
        .trim()
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '_') // Replace non-alphanumeric with underscore
        .replace(/^_+|_+$/g, ''); // Remove leading/trailing underscores
}

/**
 * Converts a snake_case string to Title Case for display.
 * Replaces underscores with spaces and capitalizes words.
 * Example: "my_node_name" -> "My Node Name"
 */
export function toTitleCase(str: string): string {
    if (!str) return '';
    return str
        .split('_')
        .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
        .join(' ');
}
