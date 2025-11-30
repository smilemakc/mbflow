# LLM Workflow Examples

Real-world workflow examples using the LLM and FunctionCall executors.

## Table of Contents

1. [Simple Q&A](#1-simple-qa)
2. [Data Extraction with Structured Outputs](#2-data-extraction-with-structured-outputs)
3. [Multi-Step Analysis](#3-multi-step-analysis)
4. [Function Calling Workflow](#4-function-calling-workflow)
5. [Image Analysis](#5-image-analysis)
6. [Content Moderation](#6-content-moderation)
7. [Translation Pipeline](#7-translation-pipeline)
8. [Code Review Assistant](#8-code-review-assistant)

## 1. Simple Q&A

Basic question-answering workflow.

```json
{
  "name": "Simple Q&A",
  "nodes": [
    {
      "id": "answer",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-3.5-turbo",
        "instruction": "You are a helpful assistant. Keep answers concise.",
        "prompt": "{{input.question}}",
        "max_tokens": 200,
        "temperature": 0.7
      }
    }
  ]
}
```

**Usage:**
```json
{"question": "What is the capital of France?"}
```

## 2. Data Extraction with Structured Outputs

Extract structured data from unstructured text.

```json
{
  "name": "Contact Info Extractor",
  "nodes": [
    {
      "id": "extract",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "prompt": "Extract contact information from this email:\n{{input.emailBody}}",
        "response_format": {
          "type": "json_schema",
          "json_schema": {
            "name": "contact_info",
            "strict": true,
            "schema": {
              "type": "object",
              "properties": {
                "name": {"type": "string"},
                "email": {"type": "string"},
                "phone": {"type": "string"},
                "company": {"type": "string"},
                "title": {"type": "string"}
              },
              "required": ["name", "email"],
              "additionalProperties": false
            }
          }
        }
      }
    }
  ]
}
```

## 3. Multi-Step Analysis

Workflow with multiple LLM nodes for complex analysis.

```json
{
  "name": "Customer Feedback Analysis",
  "nodes": [
    {
      "id": "categorize",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "prompt": "Categorize this feedback: {{input.feedback}}",
        "response_format": {
          "type": "json_schema",
          "json_schema": {
            "name": "category",
            "schema": {
              "type": "object",
              "properties": {
                "category": {
                  "type": "string",
                  "enum": ["bug", "feature_request", "praise", "complaint"]
                },
                "severity": {
                  "type": "string",
                  "enum": ["low", "medium", "high", "critical"]
                }
              },
              "required": ["category", "severity"]
            }
          }
        }
      }
    },
    {
      "id": "summarize",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-3.5-turbo",
        "prompt": "Summarize this {{input.category}} in one sentence: {{env.originalFeedback}}",
        "max_tokens": 100
      }
    },
    {
      "id": "suggest_action",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "instruction": "You are a product manager. Suggest actionable next steps.",
        "prompt": "Category: {{input.category}}\nSeverity: {{input.severity}}\nSummary: {{env.summary}}\n\nWhat should we do?",
        "max_tokens": 200
      }
    }
  ],
  "edges": [
    {"from": "categorize", "to": "summarize"},
    {"from": "summarize", "to": "suggest_action"}
  ]
}
```

## 4. Function Calling Workflow

Complete workflow with function calling.

```json
{
  "name": "Smart Assistant",
  "nodes": [
    {
      "id": "llm1",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "instruction": "You are a helpful assistant. Use functions when needed.",
        "prompt": "{{input.userMessage}}",
        "tools": [
          {
            "type": "function",
            "function": {
              "name": "get_weather",
              "description": "Get current weather for a city",
              "parameters": {
                "type": "object",
                "properties": {
                  "location": {"type": "string"},
                  "unit": {"type": "string", "enum": ["celsius", "fahrenheit"]}
                },
                "required": ["location"]
              }
            }
          },
          {
            "type": "function",
            "function": {
              "name": "get_current_time",
              "description": "Get current time",
              "parameters": {
                "type": "object",
                "properties": {
                  "format": {"type": "string", "enum": ["unix", "RFC3339", "iso8601"]}
                }
              }
            }
          }
        ]
      }
    },
    {
      "id": "func1",
      "type": "function_call",
      "config": {
        "function_name": "{{input.tool_calls[0].function.name}}",
        "arguments": "{{input.tool_calls[0].function.arguments}}"
      }
    },
    {
      "id": "llm2",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "prompt": "Format this data nicely for the user: {{input.result}}"
      }
    }
  ],
  "edges": [
    {"from": "llm1", "to": "func1"},
    {"from": "func1", "to": "llm2"}
  ]
}
```

## 5. Image Analysis

Analyze images using vision models.

```json
{
  "name": "Product Image Analyzer",
  "nodes": [
    {
      "id": "analyze_image",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4-vision-preview",
        "prompt": "Analyze this product image and extract: product name, color, condition, notable features",
        "image_url": ["{{input.imageUrl}}"],
        "max_tokens": 500
      }
    },
    {
      "id": "generate_description",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "prompt": "Create a compelling product description based on: {{input.content}}",
        "max_tokens": 200
      }
    }
  ],
  "edges": [
    {"from": "analyze_image", "to": "generate_description"}
  ]
}
```

## 6. Content Moderation

Multi-stage content moderation workflow.

```json
{
  "name": "Content Moderator",
  "nodes": [
    {
      "id": "initial_check",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "prompt": "Is this content safe? {{input.content}}",
        "response_format": {
          "type": "json_schema",
          "json_schema": {
            "name": "safety_check",
            "schema": {
              "type": "object",
              "properties": {
                "is_safe": {"type": "boolean"},
                "categories": {
                  "type": "array",
                  "items": {
                    "type": "string",
                    "enum": ["hate", "violence", "sexual", "harassment", "self-harm"]
                  }
                },
                "confidence": {"type": "number", "minimum": 0, "maximum": 1}
              },
              "required": ["is_safe", "categories", "confidence"]
            }
          }
        },
        "temperature": 0.0
      }
    },
    {
      "id": "detailed_analysis",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "instruction": "You are a content safety expert. Provide detailed analysis.",
        "prompt": "Analyze this flagged content: {{env.originalContent}}\nCategories: {{input.categories}}",
        "max_tokens": 300,
        "temperature": 0.2
      }
    }
  ],
  "edges": [
    {"from": "initial_check", "to": "detailed_analysis"}
  ]
}
```

## 7. Translation Pipeline

Multi-language translation workflow.

```json
{
  "name": "Translation Pipeline",
  "nodes": [
    {
      "id": "detect_language",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-3.5-turbo",
        "prompt": "What language is this text in? {{input.text}}",
        "response_format": {
          "type": "json_schema",
          "json_schema": {
            "name": "language",
            "schema": {
              "type": "object",
              "properties": {
                "language": {"type": "string"},
                "confidence": {"type": "number"}
              }
            }
          }
        }
      }
    },
    {
      "id": "translate",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "instruction": "You are a professional translator. Translate accurately while preserving tone and context.",
        "prompt": "Translate this {{input.language}} text to {{env.targetLanguage}}:\n{{env.originalText}}",
        "temperature": 0.3
      }
    },
    {
      "id": "quality_check",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "prompt": "Rate this translation quality (1-10) and suggest improvements:\nOriginal: {{env.originalText}}\nTranslation: {{input.content}}",
        "max_tokens": 200
      }
    }
  ],
  "edges": [
    {"from": "detect_language", "to": "translate"},
    {"from": "translate", "to": "quality_check"}
  ]
}
```

## 8. Code Review Assistant

Automated code review workflow.

```json
{
  "name": "Code Reviewer",
  "nodes": [
    {
      "id": "security_review",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "instruction": "You are a security-focused code reviewer. Find vulnerabilities.",
        "prompt": "Review this code for security issues:\n```{{input.language}}\n{{input.code}}\n```",
        "temperature": 0.2,
        "max_tokens": 500
      }
    },
    {
      "id": "performance_review",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "instruction": "You are a performance optimization expert.",
        "prompt": "Review this code for performance:\n```{{env.language}}\n{{env.code}}\n```",
        "temperature": 0.2,
        "max_tokens": 500
      }
    },
    {
      "id": "best_practices",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "instruction": "You are a code quality expert. Focus on best practices and maintainability.",
        "prompt": "Review for best practices:\n```{{env.language}}\n{{env.code}}\n```",
        "temperature": 0.3,
        "max_tokens": 500
      }
    },
    {
      "id": "combine_reviews",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "prompt": "Summarize these code reviews:\n\nSecurity: {{env.securityReview}}\n\nPerformance: {{env.performanceReview}}\n\nBest Practices: {{input.content}}",
        "max_tokens": 600
      }
    }
  ],
  "edges": [
    {"from": "security_review", "to": "performance_review"},
    {"from": "performance_review", "to": "best_practices"},
    {"from": "best_practices", "to": "combine_reviews"}
  ]
}
```

## Running Workflows

All these examples can be executed using the MBFlow SDK or REST API:

### Via SDK

```go
import (
    "context"
    "github.com/smilemakc/mbflow/pkg/sdk"
)

client := sdk.NewClient("http://localhost:8181")

// Create workflow
workflow, _ := client.CreateWorkflow(context.Background(), workflowJSON)

// Execute
execution, _ := client.ExecuteWorkflow(context.Background(), workflow.ID, map[string]interface{}{
    "question": "What is machine learning?",
})
```

### Via REST API

```bash
# Create workflow
curl -X POST http://localhost:8181/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @workflow.json

# Execute
curl -X POST http://localhost:8181/api/v1/workflows/{id}/execute \
  -H "Content-Type: application/json" \
  -d '{"question": "What is machine learning?"}'
```

## Next Steps

- [LLM Executor Guide](./LLM_EXECUTOR.md)
- [Function Calling Guide](./FUNCTION_CALLING.md)
- [API Reference](../api/)
