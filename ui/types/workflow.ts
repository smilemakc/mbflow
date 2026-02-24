/**
 * Workflow-specific types
 */

export interface WorkflowResource {
  resource_id: string;
  alias: string;
  access_type: 'read' | 'write' | 'admin';
  resource_name?: string;
  resource_type?: string;
}
