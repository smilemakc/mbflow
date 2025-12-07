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
  // NEW TEMPLATES WITH NEW NODES
  {
    id: "file-storage-backup",
    name: "API Response Backup",
    description: "Fetch API data and store to file storage with conditional retry",
    category: "data-processing",
    tags: ["file_storage", "api", "backup", "conditional"],
    difficulty: "intermediate",
    icon: "heroicons:folder",
    nodes: [
      {
        id: "http-1",
        type: "http",
        position: { x: 100, y: 150 },
        data: {
          label: "Fetch API Data",
          config: { url: "{{env.API_URL}}", method: "GET" },
        },
      },
      {
        id: "storage-1",
        type: "file_storage",
        position: { x: 400, y: 150 },
        data: {
          label: "Store Response",
          config: {
            action: "store",
            file_name: "backup-{{env.DATE}}.json",
            access_scope: "workflow",
            ttl: 86400,
          },
        },
      },
    ],
    edges: [{ id: "e1", source: "http-1", target: "storage-1" }],
    author: "MBFlow Team",
    version: "1.0.0",
  },
  {
    id: "telegram-alert-system",
    name: "Alert System with Telegram",
    description: "Monitor API and send Telegram alerts on conditions",
    category: "notification",
    tags: ["telegram", "conditional", "alert", "monitoring"],
    difficulty: "intermediate",
    icon: "heroicons:bell-alert",
    nodes: [
      {
        id: "http-1",
        type: "http",
        position: { x: 100, y: 150 },
        data: {
          label: "Health Check",
          config: { url: "{{env.HEALTH_URL}}", method: "GET" },
        },
      },
      {
        id: "conditional-1",
        type: "conditional",
        position: { x: 400, y: 150 },
        data: {
          label: "Check Status",
          config: { condition: "{{input.status_code}} != 200" },
        },
      },
      {
        id: "telegram-1",
        type: "telegram",
        position: { x: 700, y: 100 },
        data: {
          label: "Alert: Service Down",
          config: {
            bot_token: "{{env.TELEGRAM_BOT_TOKEN}}",
            chat_id: "{{env.ALERT_CHAT_ID}}",
            message_type: "text",
            text: "⚠️ Service is DOWN!\nURL: {{env.HEALTH_URL}}\nStatus: {{input.status_code}}",
            parse_mode: "HTML",
          },
        },
      },
      {
        id: "telegram-2",
        type: "telegram",
        position: { x: 700, y: 250 },
        data: {
          label: "Log: Service OK",
          config: {
            bot_token: "{{env.TELEGRAM_BOT_TOKEN}}",
            chat_id: "{{env.LOG_CHAT_ID}}",
            message_type: "text",
            text: "✅ Service OK",
          },
        },
      },
    ],
    edges: [
      { id: "e1", source: "http-1", target: "conditional-1" },
      { id: "e2", source: "conditional-1", target: "telegram-1", sourceHandle: "true" },
      { id: "e3", source: "conditional-1", target: "telegram-2", sourceHandle: "false" },
    ],
    author: "MBFlow Team",
    version: "1.0.0",
  },
  {
    id: "multi-source-aggregator",
    name: "Multi-Source Data Aggregator",
    description: "Fetch from multiple APIs, merge results, and store",
    category: "data-processing",
    tags: ["merge", "file_storage", "api", "aggregation"],
    difficulty: "intermediate",
    icon: "heroicons:arrows-pointing-in",
    nodes: [
      {
        id: "http-1",
        type: "http",
        position: { x: 100, y: 100 },
        data: { label: "Source A", config: { url: "{{env.SOURCE_A}}", method: "GET" } },
      },
      {
        id: "http-2",
        type: "http",
        position: { x: 100, y: 200 },
        data: { label: "Source B", config: { url: "{{env.SOURCE_B}}", method: "GET" } },
      },
      {
        id: "http-3",
        type: "http",
        position: { x: 100, y: 300 },
        data: { label: "Source C", config: { url: "{{env.SOURCE_C}}", method: "GET" } },
      },
      {
        id: "merge-1",
        type: "merge",
        position: { x: 400, y: 200 },
        data: { label: "Merge All", config: { merge_strategy: "all" } },
      },
      {
        id: "storage-1",
        type: "file_storage",
        position: { x: 700, y: 200 },
        data: {
          label: "Save Aggregated",
          config: { action: "store", file_name: "aggregated.json", access_scope: "result" },
        },
      },
    ],
    edges: [
      { id: "e1", source: "http-1", target: "merge-1" },
      { id: "e2", source: "http-2", target: "merge-1" },
      { id: "e3", source: "http-3", target: "merge-1" },
      { id: "e4", source: "merge-1", target: "storage-1" },
    ],
    author: "MBFlow Team",
    version: "1.0.0",
  },
  {
    id: "ai-document-processor",
    name: "AI Document Processor",
    description: "Upload document, process with AI, save result and notify",
    category: "ai-automation",
    tags: ["file_storage", "llm", "telegram", "document"],
    difficulty: "advanced",
    icon: "heroicons:document-text",
    nodes: [
      {
        id: "storage-1",
        type: "file_storage",
        position: { x: 100, y: 150 },
        data: { label: "Get Document", config: { action: "get", file_id: "{{env.DOC_ID}}" } },
      },
      {
        id: "llm-1",
        type: "llm",
        position: { x: 400, y: 150 },
        data: {
          label: "Analyze Document",
          config: {
            provider: "openai",
            model: "gpt-4",
            prompt: "Summarize this document:\n\n{{input.content}}",
          },
        },
      },
      {
        id: "storage-2",
        type: "file_storage",
        position: { x: 700, y: 100 },
        data: {
          label: "Save Summary",
          config: { action: "store", file_name: "summary.txt", access_scope: "result" },
        },
      },
      {
        id: "telegram-1",
        type: "telegram",
        position: { x: 700, y: 250 },
        data: {
          label: "Notify Complete",
          config: {
            bot_token: "{{env.TELEGRAM_BOT_TOKEN}}",
            chat_id: "{{env.NOTIFY_CHAT}}",
            message_type: "text",
            text: "📄 Document processed!\n\n{{input.content}}",
          },
        },
      },
    ],
    edges: [
      { id: "e1", source: "storage-1", target: "llm-1" },
      { id: "e2", source: "llm-1", target: "storage-2" },
      { id: "e3", source: "llm-1", target: "telegram-1" },
    ],
    author: "MBFlow Team",
    version: "1.0.0",
  },
  {
    id: "function-call-pipeline",
    name: "Function Call Pipeline",
    description: "Execute custom functions in sequence with error handling",
    category: "automation",
    tags: ["function_call", "conditional", "advanced"],
    difficulty: "advanced",
    icon: "heroicons:command-line",
    nodes: [
      {
        id: "func-1",
        type: "function_call",
        position: { x: 100, y: 150 },
        data: {
          label: "Validate Input",
          config: { function_name: "validate_input", arguments: { data: "{{env.INPUT_DATA}}" } },
        },
      },
      {
        id: "conditional-1",
        type: "conditional",
        position: { x: 400, y: 150 },
        data: { label: "Is Valid?", config: { condition: "{{input.valid}} == true" } },
      },
      {
        id: "func-2",
        type: "function_call",
        position: { x: 700, y: 100 },
        data: { label: "Process Data", config: { function_name: "process_data" } },
      },
      {
        id: "func-3",
        type: "function_call",
        position: { x: 700, y: 250 },
        data: { label: "Log Error", config: { function_name: "log_error", arguments: { error: "{{input.error}}" } } },
      },
    ],
    edges: [
      { id: "e1", source: "func-1", target: "conditional-1" },
      { id: "e2", source: "conditional-1", target: "func-2", sourceHandle: "true" },
      { id: "e3", source: "conditional-1", target: "func-3", sourceHandle: "false" },
    ],
    author: "MBFlow Team",
    version: "1.0.0",
  },
  {
    id: "telegram-file-sender",
    name: "Telegram File Sender",
    description: "Generate report, store and send via Telegram",
    category: "notification",
    tags: ["telegram", "file_storage", "report"],
    difficulty: "intermediate",
    icon: "heroicons:paper-airplane",
    nodes: [
      {
        id: "http-1",
        type: "http",
        position: { x: 100, y: 150 },
        data: { label: "Generate Report", config: { url: "{{env.REPORT_API}}", method: "POST" } },
      },
      {
        id: "storage-1",
        type: "file_storage",
        position: { x: 400, y: 150 },
        data: {
          label: "Store Report",
          config: { action: "store", file_name: "report.pdf", access_scope: "result" },
        },
      },
      {
        id: "telegram-1",
        type: "telegram",
        position: { x: 700, y: 150 },
        data: {
          label: "Send to Telegram",
          config: {
            bot_token: "{{env.TELEGRAM_BOT_TOKEN}}",
            chat_id: "{{env.REPORT_CHAT}}",
            message_type: "document",
            file_source: "url",
            file_data: "{{input.file_url}}",
          },
        },
      },
    ],
    edges: [
      { id: "e1", source: "http-1", target: "storage-1" },
      { id: "e2", source: "storage-1", target: "telegram-1" },
    ],
    author: "MBFlow Team",
    version: "1.0.0",
  },
  {
    id: "conditional-workflow-router",
    name: "Conditional Workflow Router",
    description: "Route requests based on type with different processing paths",
    category: "automation",
    tags: ["conditional", "transform", "routing"],
    difficulty: "intermediate",
    icon: "heroicons:code-bracket",
    nodes: [
      {
        id: "transform-1",
        type: "transform",
        position: { x: 100, y: 200 },
        data: {
          label: "Parse Request",
          config: { language: "jq", expression: ".type" },
        },
      },
      {
        id: "conditional-1",
        type: "conditional",
        position: { x: 400, y: 200 },
        data: { label: "Check Type", config: { condition: "{{input.result}} == \"urgent\"" } },
      },
      {
        id: "http-1",
        type: "http",
        position: { x: 700, y: 100 },
        data: { label: "Urgent Handler", config: { url: "{{env.URGENT_API}}", method: "POST" } },
      },
      {
        id: "http-2",
        type: "http",
        position: { x: 700, y: 300 },
        data: { label: "Normal Handler", config: { url: "{{env.NORMAL_API}}", method: "POST" } },
      },
    ],
    edges: [
      { id: "e1", source: "transform-1", target: "conditional-1" },
      { id: "e2", source: "conditional-1", target: "http-1", sourceHandle: "true" },
      { id: "e3", source: "conditional-1", target: "http-2", sourceHandle: "false" },
    ],
    author: "MBFlow Team",
    version: "1.0.0",
  },
  {
    id: "image-processing-pipeline",
    name: "Image Processing Pipeline",
    description: "Upload image, process with AI, store result",
    category: "ai-automation",
    tags: ["file_storage", "llm", "image", "ai"],
    difficulty: "advanced",
    icon: "heroicons:photo",
    nodes: [
      {
        id: "storage-1",
        type: "file_storage",
        position: { x: 100, y: 150 },
        data: { label: "Get Image", config: { action: "get", file_id: "{{env.IMAGE_ID}}" } },
      },
      {
        id: "llm-1",
        type: "llm",
        position: { x: 400, y: 150 },
        data: {
          label: "Analyze Image",
          config: {
            provider: "openai",
            model: "gpt-4-vision-preview",
            prompt: "Describe this image in detail",
          },
        },
      },
      {
        id: "storage-2",
        type: "file_storage",
        position: { x: 700, y: 150 },
        data: {
          label: "Save Description",
          config: { action: "store", file_name: "description.txt", access_scope: "result" },
        },
      },
    ],
    edges: [
      { id: "e1", source: "storage-1", target: "llm-1" },
      { id: "e2", source: "llm-1", target: "storage-2" },
    ],
    author: "MBFlow Team",
    version: "1.0.0",
  },
  {
    id: "merge-and-notify",
    name: "Merge Results and Notify",
    description: "Combine results from parallel tasks and send notification",
    category: "notification",
    tags: ["merge", "telegram", "parallel"],
    difficulty: "intermediate",
    icon: "heroicons:arrows-pointing-in",
    nodes: [
      {
        id: "llm-1",
        type: "llm",
        position: { x: 100, y: 100 },
        data: { label: "Task A", config: { provider: "openai", model: "gpt-4", prompt: "{{env.TASK_A}}" } },
      },
      {
        id: "llm-2",
        type: "llm",
        position: { x: 100, y: 250 },
        data: { label: "Task B", config: { provider: "openai", model: "gpt-4", prompt: "{{env.TASK_B}}" } },
      },
      {
        id: "merge-1",
        type: "merge",
        position: { x: 400, y: 175 },
        data: { label: "Merge Results", config: { merge_strategy: "all" } },
      },
      {
        id: "transform-1",
        type: "transform",
        position: { x: 700, y: 175 },
        data: {
          label: "Format Summary",
          config: { language: "jq", expression: "\"Task A: \" + .[0] + \"\\nTask B: \" + .[1]" },
        },
      },
      {
        id: "telegram-1",
        type: "telegram",
        position: { x: 1000, y: 175 },
        data: {
          label: "Send Summary",
          config: {
            bot_token: "{{env.TELEGRAM_BOT_TOKEN}}",
            chat_id: "{{env.NOTIFY_CHAT}}",
            message_type: "text",
            text: "📊 Results:\n{{input.result}}",
          },
        },
      },
    ],
    edges: [
      { id: "e1", source: "llm-1", target: "merge-1" },
      { id: "e2", source: "llm-2", target: "merge-1" },
      { id: "e3", source: "merge-1", target: "transform-1" },
      { id: "e4", source: "transform-1", target: "telegram-1" },
    ],
    author: "MBFlow Team",
    version: "1.0.0",
  },
  {
    id: "file-cleanup-workflow",
    name: "File Cleanup Workflow",
    description: "List old files, delete expired ones, and log actions",
    category: "maintenance",
    tags: ["file_storage", "conditional", "cleanup"],
    difficulty: "intermediate",
    icon: "heroicons:trash",
    nodes: [
      {
        id: "storage-1",
        type: "file_storage",
        position: { x: 100, y: 150 },
        data: { label: "List Files", config: { action: "list", limit: 100 } },
      },
      {
        id: "transform-1",
        type: "transform",
        position: { x: 400, y: 150 },
        data: {
          label: "Filter Expired",
          config: { language: "jq", expression: "[.files[] | select(.expires_at < now)]" },
        },
      },
      {
        id: "conditional-1",
        type: "conditional",
        position: { x: 700, y: 150 },
        data: { label: "Has Expired?", config: { condition: "len({{input.result}}) > 0" } },
      },
      {
        id: "func-1",
        type: "function_call",
        position: { x: 1000, y: 100 },
        data: {
          label: "Delete Files",
          config: { function_name: "batch_delete", arguments: { files: "{{input.result}}" } },
        },
      },
      {
        id: "telegram-1",
        type: "telegram",
        position: { x: 1000, y: 250 },
        data: {
          label: "No Files to Clean",
          config: {
            bot_token: "{{env.TELEGRAM_BOT_TOKEN}}",
            chat_id: "{{env.LOG_CHAT}}",
            message_type: "text",
            text: "✨ No expired files to clean",
          },
        },
      },
    ],
    edges: [
      { id: "e1", source: "storage-1", target: "transform-1" },
      { id: "e2", source: "transform-1", target: "conditional-1" },
      { id: "e3", source: "conditional-1", target: "func-1", sourceHandle: "true" },
      { id: "e4", source: "conditional-1", target: "telegram-1", sourceHandle: "false" },
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
