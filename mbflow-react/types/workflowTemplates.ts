/**
 * Workflow template types for React application
 * Ported from Vue: /mbflow-ui/src/types/workflowTemplate.ts
 */

import type { Node, Edge } from 'reactflow';

export type TemplateCategory =
  | "data-processing"
  | "api-integration"
  | "ai-automation"
  | "notification"
  | "monitoring"
  | "automation"
  | "maintenance"
  | "telegram-bots"
  | "etl"
  | "other";

export type TemplateDifficulty = "beginner" | "intermediate" | "advanced";

export interface TemplateVariable {
  name: string;
  description: string;
  type: "string" | "number" | "boolean" | "object";
  required: boolean;
  default?: any;
  placeholder?: string;
}

export interface WorkflowTemplate {
  id: string;
  name: string;
  description: string;
  category: TemplateCategory;
  tags: string[];
  difficulty: TemplateDifficulty;
  icon?: string;

  // Template content
  nodes: Node[];
  edges: Edge[];
  variables?: Record<string, TemplateVariable>;

  // Metadata
  author?: string;
  version?: string;
  created_at?: string;
  updated_at?: string;
  usage_count?: number;
}

export interface TemplateFilter {
  category?: TemplateCategory;
  difficulty?: TemplateDifficulty;
  search?: string;
  tags?: string[];
}

export interface TemplateListResponse {
  templates: WorkflowTemplate[];
  total: number;
}
