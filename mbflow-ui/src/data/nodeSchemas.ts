// NodeType import removed as it was unused

export interface OutputFieldSchema {
    type: string; // e.g., "string", "number", "object", "any"
    description?: string;
}

export type NodeOutputSchema = Record<string, OutputFieldSchema | string>;

export const NODE_OUTPUT_SCHEMAS: Record<string, NodeOutputSchema> = {
    http: {
        status: { type: "number", description: "HTTP Status Code" },
        body: { type: "any", description: "Response Body" },
        headers: { type: "object", description: "Response Headers" },
    },
    llm: {
        content: { type: "string", description: "Generated text content" },
        role: { type: "string", description: "Assistant role" },
        usage: {
            type: "object",
            description: "Token usage stats (prompt_tokens, completion_tokens)",
        },
        // For reasoning models
        reasoning_content: {
            type: "string",
            description: "Chain of thought reasoning",
        },
    },
    transform: {
        result: { type: "any", description: "Transformation result" },
    },
    function_call: {
        result: { type: "any", description: "Function execution result" },
    },
    telegram: {
        message_id: { type: "number", description: "Sent message ID" },
        chat: { type: "object", description: "Chat information" },
    },
    conditional: {
        result: { type: "boolean", description: "Condition evaluation result" },
    },
    merge: {
        output: { type: "any", description: "Merged output from executed branch" },
    },
};

export const GLOBAL_ENV_VARS = {
    // Add global env vars here if needed, or fetch from store
};
