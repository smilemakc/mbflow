import * as yup from "yup";

/**
 * Workflow validation schemas
 */

export const workflowCreateSchema = yup.object({
  name: yup
    .string()
    .required("Workflow name is required")
    .min(3, "Workflow name must be at least 3 characters")
    .max(100, "Workflow name must not exceed 100 characters")
    .matches(
      /^[a-zA-Z0-9\s\-_]+$/,
      "Workflow name can only contain letters, numbers, spaces, hyphens, and underscores",
    ),
  description: yup
    .string()
    .max(500, "Description must not exceed 500 characters")
    .optional(),
  status: yup
    .string()
    .oneOf(["draft", "active", "inactive"], "Invalid status")
    .optional()
    .default("draft"),
});

export const workflowUpdateSchema = yup.object({
  name: yup
    .string()
    .min(3, "Workflow name must be at least 3 characters")
    .max(100, "Workflow name must not exceed 100 characters")
    .matches(
      /^[a-zA-Z0-9\s\-_]+$/,
      "Workflow name can only contain letters, numbers, spaces, hyphens, and underscores",
    )
    .optional(),
  description: yup
    .string()
    .max(500, "Description must not exceed 500 characters")
    .optional(),
  status: yup
    .string()
    .oneOf(["draft", "active", "inactive"], "Invalid status")
    .optional(),
});

export type WorkflowCreateFormData = yup.InferType<typeof workflowCreateSchema>;
export type WorkflowUpdateFormData = yup.InferType<typeof workflowUpdateSchema>;
