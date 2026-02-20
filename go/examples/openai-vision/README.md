# OpenAI Vision Example

This example demonstrates how to analyze images using GPT-4o Vision through a workflow pipeline.

## Pipeline

```
HTTP Node (fetch image) → LLM Node (GPT-4o vision analysis)
```

## Features

- **HTTP Node**: Fetches an image from httpbin.org, auto-converts to base64
- **LLM Node**: Sends image to GPT-4o for vision analysis
- **Template Resolution**: Uses `{{input.body_base64}}` to pass image data between nodes

## Usage

```bash
# Set your OpenAI API key
export OPENAI_API_KEY=sk-...

# Run the example
go run main.go
```

## Expected Output

```
=== Step 1: Fetching image from httpbin.org ===
Status: 200
Content-Type: image/jpeg
Image size: 35588 bytes
Has base64: true

=== Step 2: Analyzing image with GPT-4o Vision ===

=== GPT-4o Vision Response ===
The image shows a pig standing on a green grassy hill...

=== Token Usage ===
Prompt tokens: 1273
Completion tokens: 87
Total tokens: 1360
```

## Workflow Configuration

```json
{
  "nodes": [
    {
      "id": "fetch_image",
      "type": "http",
      "config": {
        "method": "GET",
        "url": "https://httpbin.org/image/jpeg"
      }
    },
    {
      "id": "analyze_image",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4o",
        "prompt": "Describe this image in detail.",
        "files": [{
          "data": "{{input.body_base64}}",
          "mime_type": "{{input.content_type}}",
          "name": "image.jpg"
        }]
      }
    }
  ],
  "edges": [
    {"from": "fetch_image", "to": "analyze_image"}
  ]
}
```

## Supported Image Formats

- JPEG (`image/jpeg`)
- PNG (`image/png`)  
- GIF (`image/gif`)
- WebP (`image/webp`)
- PDF (`application/pdf`)

## Notes

- The HTTP node auto-detects binary content and returns `body_base64`
- Vision is available in `gpt-4o`, `gpt-4o-mini`, and `gpt-4-turbo` models
- Large images consume more tokens
