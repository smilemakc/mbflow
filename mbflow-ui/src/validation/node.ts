import * as yup from "yup";

/**
 * Node configuration validation schemas
 */

// HTTP Node validation
export const httpNodeConfigSchema = yup.object({
  url: yup
    .string()
    .required("URL is required")
    .url("Must be a valid URL")
    .max(2000, "URL must not exceed 2000 characters"),
  method: yup
    .string()
    .oneOf(
      ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"],
      "Invalid HTTP method",
    )
    .required("HTTP method is required")
    .default("GET"),
  headers: yup.object().optional().default({}),
  body: yup
    .string()
    .optional()
    .test("valid-json", "Body must be valid JSON", (value) => {
      if (!value) return true;
      try {
        JSON.parse(value);
        return true;
      } catch {
        return false;
      }
    }),
  timeout: yup
    .number()
    .optional()
    .min(1, "Timeout must be at least 1 second")
    .max(300, "Timeout must not exceed 300 seconds")
    .default(30),
  retry_count: yup
    .number()
    .optional()
    .min(0, "Retry count cannot be negative")
    .max(10, "Retry count must not exceed 10")
    .default(0),
});

// LLM Node validation
export const llmNodeConfigSchema = yup.object({
  provider: yup
    .string()
    .oneOf(["openai", "anthropic", "google", "ollama"], "Invalid LLM provider")
    .required("Provider is required"),
  model: yup
    .string()
    .required("Model is required")
    .min(1, "Model name is required"),
  prompt: yup
    .string()
    .required("Prompt is required")
    .min(1, "Prompt cannot be empty")
    .max(10000, "Prompt must not exceed 10000 characters"),
  temperature: yup
    .number()
    .optional()
    .min(0, "Temperature must be between 0 and 2")
    .max(2, "Temperature must be between 0 and 2")
    .default(0.7),
  max_tokens: yup
    .number()
    .optional()
    .min(1, "Max tokens must be at least 1")
    .max(100000, "Max tokens must not exceed 100000")
    .default(1000),
  api_key: yup.string().optional().min(1, "API key cannot be empty"),
  system_message: yup
    .string()
    .optional()
    .max(5000, "System message must not exceed 5000 characters"),
});

// Transform Node validation
export const transformNodeConfigSchema = yup.object({
  transform_type: yup
    .string()
    .oneOf(["jq", "javascript", "template"], "Invalid transform type")
    .required("Transform type is required")
    .default("jq"),
  expression: yup
    .string()
    .required("Expression is required")
    .min(1, "Expression cannot be empty")
    .max(10000, "Expression must not exceed 10000 characters"),
  output_key: yup
    .string()
    .optional()
    .matches(
      /^[a-zA-Z_][a-zA-Z0-9_]*$/,
      "Output key must be a valid identifier (letters, numbers, underscores, cannot start with number)",
    ),
});

// Conditional Node validation
export const conditionalNodeConfigSchema = yup.object({
  condition_type: yup
    .string()
    .oneOf(["expression", "comparison"], "Invalid condition type")
    .required("Condition type is required")
    .default("expression"),
  expression: yup.string().when("condition_type", {
    is: "expression",
    then: (schema) => schema.required("Expression is required"),
    otherwise: (schema) => schema.optional(),
  }),
  left_operand: yup.string().when("condition_type", {
    is: "comparison",
    then: (schema) => schema.required("Left operand is required"),
    otherwise: (schema) => schema.optional(),
  }),
  operator: yup.string().when("condition_type", {
    is: "comparison",
    then: (schema) =>
      schema
        .oneOf(
          [
            "==",
            "!=",
            ">",
            "<",
            ">=",
            "<=",
            "contains",
            "startsWith",
            "endsWith",
          ],
          "Invalid operator",
        )
        .required("Operator is required"),
    otherwise: (schema) => schema.optional(),
  }),
  right_operand: yup.string().when("condition_type", {
    is: "comparison",
    then: (schema) => schema.required("Right operand is required"),
    otherwise: (schema) => schema.optional(),
  }),
});

// Merge Node validation
export const mergeNodeConfigSchema = yup.object({
  merge_strategy: yup
    .string()
    .oneOf(["first", "last", "all", "custom"], "Invalid merge strategy")
    .required("Merge strategy is required")
    .default("all"),
  merge_key: yup
    .string()
    .optional()
    .matches(
      /^[a-zA-Z_][a-zA-Z0-9_]*$/,
      "Merge key must be a valid identifier",
    ),
});

// Base node validation (common fields)
export const baseNodeSchema = yup.object({
  name: yup
    .string()
    .required("Node name is required")
    .min(1, "Node name cannot be empty")
    .max(100, "Node name must not exceed 100 characters")
    .matches(
      /^[a-zA-Z0-9\s\-_]+$/,
      "Node name can only contain letters, numbers, spaces, hyphens, and underscores",
    ),
  description: yup
    .string()
    .optional()
    .max(500, "Description must not exceed 500 characters"),
});

// Export type inference
export type HTTPNodeConfigFormData = yup.InferType<typeof httpNodeConfigSchema>;
export type LLMNodeConfigFormData = yup.InferType<typeof llmNodeConfigSchema>;
export type TransformNodeConfigFormData = yup.InferType<
  typeof transformNodeConfigSchema
>;
export type ConditionalNodeConfigFormData = yup.InferType<
  typeof conditionalNodeConfigSchema
>;
export type MergeNodeConfigFormData = yup.InferType<
  typeof mergeNodeConfigSchema
>;
export type BaseNodeFormData = yup.InferType<typeof baseNodeSchema>;
