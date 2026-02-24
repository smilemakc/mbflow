import { useMemo } from 'react';

/**
 * Helper for converting array to string and back.
 * Used for stop_sequences, tags, and other array-like text inputs.
 *
 * @param array - Array to convert to string
 * @param separator - Separator to join array elements (default: '\n')
 * @returns Tuple of [string value, parser function]
 *
 * @example
 * // In component state
 * const [stopSequences, setStopSequences] = useState(config.stop_sequences || []);
 * const [text, parseText] = useArrayToString(stopSequences, '\n');
 *
 * // In JSX
 * <textarea
 *   value={text}
 *   onChange={(e) => {
 *     const newArray = parseText(e.target.value);
 *     setStopSequences(newArray);
 *     updateConfig({ stop_sequences: newArray });
 *   }}
 * />
 */
export function useArrayToString(
  array: string[] | undefined,
  separator: string = '\n'
): [string, (str: string) => string[]] {
  const stringValue = useMemo(() => {
    if (!array || array.length === 0) return '';
    return array.join(separator);
  }, [array, separator]);

  const parseString = (str: string): string[] => {
    if (!str.trim()) return [];
    return str
      .split(separator)
      .map((s) => s.trim())
      .filter((s) => s.length > 0);
  };

  return [stringValue, parseString];
}
