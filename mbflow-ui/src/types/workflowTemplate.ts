/**
 * Workflow template types
 */

import type { Node, Edge } from "@vue-flow/core";

export interface WorkflowTemplate {
  id: string;
  name: string;
  description: string;
  category: TemplateCategory;
  tags: string[];
  difficulty: "beginner" | "intermediate" | "advanced";
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

export type TemplateCategory =
  | "data-processing"
  | "api-integration"
  | "ai-automation"
  | "notification"
  | "monitoring"
  | "etl"
  | "other";

export interface TemplateVariable {
  name: string;
  description: string;
  type: "string" | "number" | "boolean" | "object";
  required: boolean;
  default?: any;
  placeholder?: string;
}

export interface TemplateFilter {
  category?: TemplateCategory;
  difficulty?: "beginner" | "intermediate" | "advanced";
  search?: string;
  tags?: string[];
}
