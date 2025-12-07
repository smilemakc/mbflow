/**
 * Generate a node ID based on node type and existing nodes
 * Pattern: nodeType or nodeType_2, nodeType_3, etc.
 *
 * @param nodeType - The type of node (e.g., 'http', 'llm')
 * @param existingIds - Array of existing node IDs to check for duplicates
 * @returns Generated node ID
 */
export function generateNodeId(
  nodeType: string,
  existingIds: string[],
): string {
  // Check if base nodeType is available
  if (!existingIds.includes(nodeType)) {
    return nodeType;
  }

  // Find the next available index
  let index = 2;
  while (existingIds.includes(`${nodeType}_${index}`)) {
    index++;
  }

  return `${nodeType}_${index}`;
}

/**
 * Validate node ID
 * Only allows letters (a-Z) and underscores (_)
 *
 * @param id - Node ID to validate
 * @returns true if valid, false otherwise
 */
export function validateNodeId(id: string): boolean {
  if (!id || id.trim() === "") {
    return false;
  }

  // Only allow a-Z and _
  const validPattern = /^[a-zA-Z_1-9\-]+$/;
  return validPattern.test(id);
}

/**
 * Check if node ID is unique among existing nodes
 *
 * @param id - Node ID to check
 * @param existingIds - Array of existing node IDs
 * @param currentId - Current node ID (for editing existing nodes)
 * @returns true if unique, false otherwise
 */
export function isNodeIdUnique(
  id: string,
  existingIds: string[],
  currentId?: string,
): boolean {
  // If editing, allow the current ID
  if (currentId && id === currentId) {
    return true;
  }

  return !existingIds.includes(id);
}
