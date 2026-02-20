# OpenAI Responses API Basic Example

This example demonstrates how to use the OpenAI Responses API with MBFlow.

## What is the Responses API?

The Responses API is OpenAI's newest API that combines the best of Chat Completions and Assistants APIs. It's designed for:
- GPT-5, o3-mini, gpt-4.1+ and newer reasoning models
- Multi-turn conversations with persistent reasoning state
- Hosted tools (web search, file search, code interpreter)
- Polymorphic output items (messages, tool calls, web searches)

## Setup

1. Set your OpenAI API key:
```bash
export OPENAI_API_KEY="sk-..."
```

2. Build the example:
```bash
go build -o bin/responses_api_basic ./examples/responses_api_basic
```

3. Run the example:
```bash
./bin/responses_api_basic
```

## Expected Output

```
=== OpenAI Responses API Basic Example ===

Workflow: responses-demo
Node: Story Generator (provider: openai-responses, model: gpt-4.1)

Executing workflow...

=== Results ===

Story:
In a peaceful grove beneath a silver moon, a unicorn named Lumina discovered a hidden pool that reflected the stars. As she dipped her horn into the water, the pool began to shimmer, revealing a pathway to a magical realm of endless night skies. Filled with wonder, Lumina whispered a wish for all who dream to find their own hidden magic, and as she glanced back, her hoofprints sparkled like stardust.

Usage:
  Prompt tokens: 36
  Completion tokens: 87
  Total tokens: 123

Status: completed

Output Items: 1
  [0] Type: message, Status: completed
```

## Key Features Demonstrated

1. **Provider Selection**: Using `openai-responses` provider instead of `openai`
2. **Builder API**: Using `NewOpenAIResponsesNode` for type-safe configuration
3. **Response Structure**: Accessing both legacy fields (`content`) and new fields (`status`, `output_items`)

## Next Steps

See other examples:
- `responses_api_web_search` - Using web search tool
- `responses_api_reasoning` - Using reasoning models (o3-mini)
