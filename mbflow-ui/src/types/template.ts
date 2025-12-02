export interface VariableContext {
  workflowVars: Record<string, any>;
  executionVars: Record<string, any>;
  inputVars: Record<string, any>;
}

export interface TemplateOptions {
  strictMode?: boolean;
  placeholderOnMissing?: boolean;
}

export type VariableType = "env" | "input";

export interface ParsedVariable {
  fullMatch: string;
  type: VariableType;
  path: string;
}
