# OpenAI Function Calling Demo

This example demonstrates the complete OpenAI function calling flow with conversation continuation in MBFlow.

## Features

- Define functions that OpenAI can call
- Let OpenAI decide when to call functions
- Execute functions with extracted parameters
- Continue the conversation after function execution
- Support for multiple function handlers (script, builtin, http)

## How it Works

1. **Define Functions**: Create function definitions with JSON Schema parameters
2. **AI Decision**: OpenAI analyzes the prompt and decides which function to call
3. **Parameter Extraction**: OpenAI extracts the required parameters from context
4. **Function Execution**: The `FunctionCallExecutor` executes the function with the parameters
5. **Continue Conversation**: The `OpenAIFunctionResultExecutor` sends the function result back to OpenAI
6. **Final Response**: OpenAI formulates a natural language response using the function result

## Workflow Structure

```
start
  ↓
ask_ai (OpenAI with function calling)
  ↓
execute_function (FunctionCallExecutor)
  ↓
continue_conversation (OpenAIFunctionResultExecutor)
  ↓
end
```

## Running the Example

```bash
export OPENAI_API_KEY=your_api_key_here
go run examples/function-calling-demo/main.go
```

## Function Handlers

### Script Handler

Executes functions using `expr-lang` expressions:

```go
AddNodeWithConfig(string(mbflow.NodeTypeFunctionCall), "execute_function", &mbflow.FunctionCallConfig{
    InputKey: "ai_response",
    Handler:  "script",
    HandlerConfig: map[string]interface{}{
        "script": `{
            "temperature": 22,
            "unit": unit != nil ? unit : "celsius",
            "location": location
        }`,
    },
    OutputKey: "function_result",
})
```

### Builtin Handler

For pre-defined Go functions (extend `executeBuiltin` method).

### HTTP Handler

For calling external APIs (coming soon).

## Configuration

### OpenAICompletionConfig with Tools

```go
&mbflow.OpenAICompletionConfig{
    Model:      "gpt-4o",
    Prompt:     "Your prompt here",
    Tools:      []mbflow.OpenAITool{...},
    ToolChoice: "auto", // or "none", or specific function
}
```

### Function Tool Definition

```go
mbflow.OpenAITool{
    Type: "function",
    Function: mbflow.OpenAIFunction{
        Name:        "function_name",
        Description: "What this function does",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "param1": map[string]interface{}{
                    "type":        "string",
                    "description": "Parameter description",
                },
            },
            "required": []string{"param1"},
        },
    },
}
```

## Output

The workflow returns:
- `ai_response`: Initial OpenAI response including `tool_calls`
- `function_result`: Result from executing the function
- `final_response`: Natural language response from OpenAI after processing the function result

## New Node Type: `OpenAIFunctionResultExecutor`

This executor continues the conversation after function execution:

```go
AddNodeWithConfig(string(mbflow.NodeTypeOpenAIFunctionResult), "continue_conversation", &mbflow.OpenAIFunctionResponseConfig{
    AIResponseKey:     "ai_response",      // Key containing the original AI response
    FunctionResultKey: "function_result",  // Key containing the function result
    Model:             "gpt-4o",
    MaxTokens:         300,
    OutputKey:         "final_response",
    APIKey:            apiKey,
})
```

The executor:
1. Extracts the tool_calls from the original AI response
2. Builds a message history with the assistant's tool call
3. Adds a tool message with the function result
4. Sends the complete history back to OpenAI
5. Returns the AI's natural language response

## Use Cases

- **Weather APIs**: Ask about weather, AI calls weather API and formats the response naturally
- **Database Queries**: Natural language to SQL queries with formatted results
- **Calculator Functions**: Math operations with explanations
- **Data Lookups**: Search and retrieve information with context
- **External Integrations**: Call third-party services and present results conversationally
