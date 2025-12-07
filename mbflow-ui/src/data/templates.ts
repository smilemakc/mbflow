/**
 * Predefined workflow templates
 */

import type { WorkflowTemplate } from "@/types/workflowTemplate";

export const workflowTemplates: WorkflowTemplate[] = [
  {
    id: "http-to-llm",
    name: "API to AI Processing",
    description:
      "Fetch data from an API, process it with an LLM, and return the result",
    category: "ai-automation",
    tags: ["api", "llm", "ai", "beginner-friendly"],
    difficulty: "beginner",
    icon: "heroicons:sparkles",
    nodes: [
      {
        id: "http-1",
        type: "http",
        position: { x: 100, y: 100 },
        data: {
          label: "Fetch Data",
          config: {
            url: "https://api.example.com/data",
            method: "GET",
            headers: {},
          },
        },
      },
      {
        id: "llm-1",
        type: "llm",
        position: { x: 400, y: 100 },
        data: {
          label: "Process with AI",
          config: {
            provider: "openai",
            model: "gpt-4",
            prompt:
              "Analyze the following data and provide insights:\\n\\n{{input.http-1.body}}",
            temperature: 0.7,
            max_tokens: 500,
          },
        },
      },
    ],
    edges: [
      {
        id: "e1",
        source: "http-1",
        target: "llm-1",
      },
    ],
    variables: {
      api_url: {
        name: "api_url",
        description: "The API endpoint to fetch data from",
        type: "string",
        required: true,
        placeholder: "https://api.example.com/data",
      },
      llm_model: {
        name: "llm_model",
        description: "LLM model to use for processing",
        type: "string",
        required: false,
        default: "gpt-4",
      },
    },
    author: "MBFlow Team",
    version: "1.0.0",
  },
  {
    id: "data-transform-pipeline",
    name: "Data Transformation Pipeline",
    description: "Fetch, transform, and store data with conditional logic",
    category: "data-processing",
    tags: ["etl", "transform", "conditional"],
    difficulty: "intermediate",
    icon: "heroicons:arrow-path",
    nodes: [
      {
        id: "http-1",
        type: "http",
        position: { x: 100, y: 150 },
        data: {
          label: "Fetch Source Data",
          config: {
            url: "{{env.SOURCE_API_URL}}",
            method: "GET",
          },
        },
      },
      {
        id: "transform-1",
        type: "transform",
        position: { x: 400, y: 150 },
        data: {
          label: "Transform Data",
          config: {
            transform_type: "jq",
            expression: ".data | map({id, name, email})",
          },
        },
      },
      {
        id: "conditional-1",
        type: "conditional",
        position: { x: 700, y: 150 },
        data: {
          label: "Check Data Quality",
          config: {
            condition_type: "expression",
            expression: "{{input.transform-1.result | length}} > 0",
          },
        },
      },
      {
        id: "http-2",
        type: "http",
        position: { x: 1000, y: 100 },
        data: {
          label: "Store Valid Data",
          config: {
            url: "{{env.DESTINATION_API_URL}}",
            method: "POST",
            body: "{{input.transform-1.result}}",
          },
        },
      },
      {
        id: "http-3",
        type: "http",
        position: { x: 1000, y: 250 },
        data: {
          label: "Log Error",
          config: {
            url: "{{env.ERROR_LOG_URL}}",
            method: "POST",
            body: JSON.stringify({ error: "No data to process" }),
          },
        },
      },
    ],
    edges: [
      { id: "e1", source: "http-1", target: "transform-1" },
      { id: "e2", source: "transform-1", target: "conditional-1" },
      {
        id: "e3",
        source: "conditional-1",
        target: "http-2",
        sourceHandle: "true",
      },
      {
        id: "e4",
        source: "conditional-1",
        target: "http-3",
        sourceHandle: "false",
      },
    ],
    author: "MBFlow Team",
    version: "1.0.0",
  },
  {
    id: "webhook-notification",
    name: "Webhook to Notification",
    description:
      "Receive webhook data and send notifications via multiple channels",
    category: "notification",
    tags: ["webhook", "notification", "slack", "email"],
    difficulty: "beginner",
    icon: "heroicons:bell-alert",
    nodes: [
      {
        id: "http-1",
        type: "http",
        position: { x: 100, y: 200 },
        data: {
          label: "Parse Webhook",
          config: {
            url: "https://webhook.site/unique-id",
            method: "POST",
          },
        },
      },
      {
        id: "transform-1",
        type: "transform",
        position: { x: 400, y: 200 },
        data: {
          label: "Format Message",
          config: {
            transform_type: "template",
            expression:
              "Alert: {{input.http-1.body.message}}\\nSeverity: {{input.http-1.body.severity}}",
          },
        },
      },
      {
        id: "http-2",
        type: "http",
        position: { x: 700, y: 150 },
        data: {
          label: "Send to Slack",
          config: {
            url: "{{env.SLACK_WEBHOOK_URL}}",
            method: "POST",
            body: JSON.stringify({
              text: "{{input.transform-1.result}}",
            }),
          },
        },
      },
      {
        id: "http-3",
        type: "http",
        position: { x: 700, y: 250 },
        data: {
          label: "Send Email",
          config: {
            url: "{{env.EMAIL_API_URL}}",
            method: "POST",
            body: JSON.stringify({
              to: "{{env.ALERT_EMAIL}}",
              subject: "System Alert",
              body: "{{input.transform-1.result}}",
            }),
          },
        },
      },
    ],
    edges: [
      { id: "e1", source: "http-1", target: "transform-1" },
      { id: "e2", source: "transform-1", target: "http-2" },
      { id: "e3", source: "transform-1", target: "http-3" },
    ],
    author: "MBFlow Team",
    version: "1.0.0",
  },
  {
    id: "ai-content-generator",
    name: "AI Content Generator",
    description: "Generate content using multiple LLM calls with refinement",
    category: "ai-automation",
    tags: ["llm", "content", "ai", "advanced"],
    difficulty: "advanced",
    icon: "heroicons:document-text",
    nodes: [
      {
        id: "llm-1",
        type: "llm",
        position: { x: 100, y: 150 },
        data: {
          label: "Generate Outline",
          config: {
            provider: "openai",
            model: "gpt-4",
            prompt: "Create a detailed outline for: {{env.TOPIC}}",
            temperature: 0.8,
          },
        },
      },
      {
        id: "llm-2",
        type: "llm",
        position: { x: 400, y: 150 },
        data: {
          label: "Write Content",
          config: {
            provider: "openai",
            model: "gpt-4",
            prompt:
              "Based on this outline:\\n{{input.llm-1.content}}\\n\\nWrite detailed content.",
            temperature: 0.7,
            max_tokens: 2000,
          },
        },
      },
      {
        id: "llm-3",
        type: "llm",
        position: { x: 700, y: 150 },
        data: {
          label: "Refine & Polish",
          config: {
            provider: "openai",
            model: "gpt-4",
            prompt: "Refine and polish this content:\\n{{input.llm-2.content}}",
            temperature: 0.5,
          },
        },
      },
      {
        id: "http-1",
        type: "http",
        position: { x: 1000, y: 150 },
        data: {
          label: "Save Content",
          config: {
            url: "{{env.CONTENT_API_URL}}",
            method: "POST",
            body: "{{input.llm-3.content}}",
          },
        },
      },
    ],
    edges: [
      { id: "e1", source: "llm-1", target: "llm-2" },
      { id: "e2", source: "llm-2", target: "llm-3" },
      { id: "e3", source: "llm-3", target: "http-1" },
    ],
    author: "MBFlow Team",
    version: "1.0.0",
  },
  {
    id: "parallel-api-calls",
    name: "Parallel API Aggregation",
    description: "Make multiple API calls in parallel and merge results",
    category: "api-integration",
    tags: ["api", "parallel", "merge", "intermediate"],
    difficulty: "intermediate",
    icon: "heroicons:arrows-right-left",
    nodes: [
      {
        id: "http-1",
        type: "http",
        position: { x: 100, y: 100 },
        data: {
          label: "Fetch Users",
          config: {
            url: "{{env.API_URL}}/users",
            method: "GET",
          },
        },
      },
      {
        id: "http-2",
        type: "http",
        position: { x: 100, y: 200 },
        data: {
          label: "Fetch Orders",
          config: {
            url: "{{env.API_URL}}/orders",
            method: "GET",
          },
        },
      },
      {
        id: "http-3",
        type: "http",
        position: { x: 100, y: 300 },
        data: {
          label: "Fetch Products",
          config: {
            url: "{{env.API_URL}}/products",
            method: "GET",
          },
        },
      },
      {
        id: "merge-1",
        type: "merge",
        position: { x: 400, y: 200 },
        data: {
          label: "Merge All Data",
          config: {
            merge_strategy: "all",
          },
        },
      },
      {
        id: "transform-1",
        type: "transform",
        position: { x: 700, y: 200 },
        data: {
          label: "Format Response",
          config: {
            transform_type: "jq",
            expression: "{users: .[0], orders: .[1], products: .[2]}",
          },
        },
      },
    ],
    edges: [
      { id: "e1", source: "http-1", target: "merge-1" },
      { id: "e2", source: "http-2", target: "merge-1" },
      { id: "e3", source: "http-3", target: "merge-1" },
      { id: "e4", source: "merge-1", target: "transform-1" },
    ],
    author: "MBFlow Team",
    version: "1.0.0",
  },
];

// Helper functions
export function getTemplateById(id: string): WorkflowTemplate | undefined {
  return workflowTemplates.find((t) => t.id === id);
}

export function getTemplatesByCategory(category: string): WorkflowTemplate[] {
  return workflowTemplates.filter((t) => t.category === category);
}

export function getTemplatesByDifficulty(
  difficulty: string,
): WorkflowTemplate[] {
  return workflowTemplates.filter((t) => t.difficulty === difficulty);
}

export function searchTemplates(query: string): WorkflowTemplate[] {
  const lowerQuery = query.toLowerCase();
  return workflowTemplates.filter(
    (t) =>
      t.name.toLowerCase().includes(lowerQuery) ||
      t.description.toLowerCase().includes(lowerQuery) ||
      t.tags.some((tag) => tag.toLowerCase().includes(lowerQuery)),
  );
}
