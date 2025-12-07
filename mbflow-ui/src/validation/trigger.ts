import * as yup from "yup";

/**
 * Trigger validation schemas
 */

// Cron expression validation (basic)
const cronRegex =
  /^(\*|([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])|\*\/([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])) (\*|([0-9]|1[0-9]|2[0-3])|\*\/([0-9]|1[0-9]|2[0-3])) (\*|([1-9]|1[0-9]|2[0-9]|3[0-1])|\*\/([1-9]|1[0-9]|2[0-9]|3[0-1])) (\*|([1-9]|1[0-2])|\*\/([1-9]|1[0-2])) (\*|([0-6])|\*\/([0-6]))$/;

// Base trigger schema
export const triggerBaseSchema = yup.object({
  workflow_id: yup
    .string()
    .required("Workflow is required")
    .uuid("Invalid workflow ID"),
  name: yup
    .string()
    .required("Trigger name is required")
    .min(3, "Trigger name must be at least 3 characters")
    .max(100, "Trigger name must not exceed 100 characters"),
  description: yup
    .string()
    .max(500, "Description must not exceed 500 characters")
    .optional(),
  type: yup
    .string()
    .oneOf(["manual", "schedule", "webhook", "event"], "Invalid trigger type")
    .required("Trigger type is required"),
  status: yup
    .string()
    .oneOf(["enabled", "disabled"], "Invalid status")
    .optional()
    .default("enabled"),
});

// Schedule config schema
export const scheduleConfigSchema = yup.object({
  cron: yup
    .string()
    .required("Cron expression is required")
    .matches(cronRegex, "Invalid cron expression format"),
  timezone: yup.string().optional().default("UTC"),
});

// Webhook config schema
export const webhookConfigSchema = yup.object({
  webhook_path: yup
    .string()
    .required("Webhook path is required")
    .matches(/^\//, "Webhook path must start with /")
    .max(200, "Webhook path must not exceed 200 characters"),
  http_method: yup
    .string()
    .oneOf(["GET", "POST", "PUT", "PATCH", "DELETE"], "Invalid HTTP method")
    .optional()
    .default("POST"),
  auth_type: yup
    .string()
    .oneOf(
      ["none", "api_key", "bearer", "basic"],
      "Invalid authentication type",
    )
    .optional()
    .default("none"),
  auth_config: yup.object().optional().default({}),
});

// Event config schema
export const eventConfigSchema = yup.object({
  event_type: yup
    .string()
    .required("Event type is required")
    .min(3, "Event type must be at least 3 characters")
    .max(100, "Event type must not exceed 100 characters")
    .matches(
      /^[a-z0-9._-]+$/,
      "Event type can only contain lowercase letters, numbers, dots, hyphens, and underscores",
    ),
  event_filter: yup.object().optional().default({}),
});

// Manual trigger (no config needed)
export const manualTriggerSchema = triggerBaseSchema.shape({
  type: yup.string().oneOf(["manual"]).required(),
  config: yup.object().optional().default({}),
});

// Schedule trigger
export const scheduleTriggerSchema = triggerBaseSchema.shape({
  type: yup.string().oneOf(["schedule"]).required(),
  config: scheduleConfigSchema.required("Schedule configuration is required"),
});

// Webhook trigger
export const webhookTriggerSchema = triggerBaseSchema.shape({
  type: yup.string().oneOf(["webhook"]).required(),
  config: webhookConfigSchema.required("Webhook configuration is required"),
});

// Event trigger
export const eventTriggerSchema = triggerBaseSchema.shape({
  type: yup.string().oneOf(["event"]).required(),
  config: eventConfigSchema.required("Event configuration is required"),
});

// Combined trigger schema that validates based on type
export const triggerCreateSchema = yup.lazy((value: any) => {
  switch (value?.type) {
    case "schedule":
      return scheduleTriggerSchema;
    case "webhook":
      return webhookTriggerSchema;
    case "event":
      return eventTriggerSchema;
    case "manual":
    default:
      return manualTriggerSchema;
  }
});

export type TriggerBaseFormData = yup.InferType<typeof triggerBaseSchema>;
export type ScheduleConfigFormData = yup.InferType<typeof scheduleConfigSchema>;
export type WebhookConfigFormData = yup.InferType<typeof webhookConfigSchema>;
export type EventConfigFormData = yup.InferType<typeof eventConfigSchema>;
