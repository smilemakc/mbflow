#!/usr/bin/env python3
"""A minimal stand-in for the Anthropic Messages API, for dry runs only.

It lets the functional suite exercise MBFlow's whole LLM path (request
format, credentials, tool-calling loop, structured output) without a real
key. Its answers are canned, so a dry run says nothing about model quality.

    python3 test/functional/fake_anthropic.py 18080 EXPECTED_KEY

Requests whose x-api-key differs from EXPECTED_KEY get a 401, like the real
API, so a dry run also checks that the key reaches the provider. Like the
real API for current models (the suite's default), it rejects sampling
parameters and, for the models that refuse it, forced tool_choice; it
answers structured output requests (output_config.format) with JSON text and
puts a thinking block before tool calls.
"""
import json
import re
import sys
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

EXPECTED_KEY = sys.argv[2] if len(sys.argv) > 2 else ""


# Rules of the real API this stand-in enforces (current models).
NO_FORCED_TOOLS = ("claude-opus-5-5", "claude-sonnet-5-5", "claude-fable-5-1", "claude-mythos-5-1")
SAMPLING_OK = ("claude-3", "claude-opus-4-0", "claude-opus-4-1", "claude-opus-4-5", "claude-opus-4-6",
               "claude-sonnet-4", "claude-haiku-4")


def invalid(body):
    model = body.get("model") or ""
    if "no-such-model" in model:
        return 404, "not_found_error", f"model: {model}"
    for param in ("temperature", "top_p", "top_k"):
        if param in body and not model.startswith(SAMPLING_OK):
            return 400, "invalid_request_error", f"`{param}` is deprecated for this model."
    if (body.get("tool_choice") or {}).get("type") in ("tool", "any") and model.startswith(NO_FORCED_TOOLS):
        return 400, "invalid_request_error", 'tool_choice: type "tool" and "any" are not supported for this model.'
    return None


def classify(prompt):
    """Structured output: classify countries by latitude."""
    countries = []
    start = prompt.find("[")
    items = json.loads(prompt[start:prompt.rindex("]") + 1]) if start >= 0 else []
    for item in items:
        lat = abs(float((item.get("latlng") or [0])[0]))
        climate = "tropical" if lat < 23.5 else ("polar" if lat > 66 else "temperate")
        countries.append({"name": item.get("name"), "climate": climate, "fun_fact": f"{item.get('name')} has a fun fact."})
    return {"countries": countries}


def text_of(content):
    if isinstance(content, str):
        return content
    return "\n".join(b.get("text", "") for b in content if b.get("type") == "text")


def tool_results(messages):
    out = []
    for m in messages:
        if isinstance(m.get("content"), list):
            out += [b for b in m["content"] if b.get("type") == "tool_result"]
    return out


def respond(body):
    messages = body.get("messages", [])
    prompt = text_of(messages[0]["content"]) if messages else ""
    tools = {t["name"]: t for t in body.get("tools", [])}
    usage = {"input_tokens": len(json.dumps(body)) // 4, "output_tokens": 20}

    if ((body.get("output_config") or {}).get("format") or {}).get("type") == "json_schema":
        return {"content": [{"type": "text", "text": json.dumps(classify(prompt))}],
                "stop_reason": "end_turn", "usage": usage}

    forced = (body.get("tool_choice") or {}).get("name")
    if forced:
        return {"content": [{"type": "tool_use", "id": "toolu_s", "name": forced, "input": classify(prompt)}],
                "stop_reason": "tool_use", "usage": usage}

    if "http_request" in tools:
        results = tool_results(messages)
        if not results:
            url = re.search(r"https?://\S+", prompt).group(0).rstrip(".,)")
            return {"content": [{"type": "thinking", "thinking": "", "signature": "fake-signature"},
                                {"type": "text", "text": "Fetching the data."},
                                {"type": "tool_use", "id": "toolu_1", "name": "http_request", "input": {"url": url, "method": "GET"}}],
                    "stop_reason": "tool_use", "usage": usage}
        payload = json.loads(results[-1]["content"])
        animals = payload.get("body", {}).get("animals", [])
        letter = re.search(r"letter '(\w)'", prompt).group(1)
        hits = [a for a in animals if a.startswith(letter)]
        return {"content": [{"type": "text", "text": f"{len(hits)}: " + ", ".join(hits)}],
                "stop_reason": "end_turn", "usage": usage}

    words = re.findall(r"[A-Za-z]{2,}", prompt)
    return {"content": [{"type": "text", "text": "Summary: " + " ".join(words[:25])}],
            "stop_reason": "end_turn", "usage": usage}


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        if self.path != "/v1/messages" or self.headers.get("anthropic-version") is None:
            return self.reply(404, {"type": "error", "error": {"type": "not_found_error", "message": self.path}})
        if self.headers.get("x-api-key") != EXPECTED_KEY:
            return self.reply(401, {"type": "error", "error": {"type": "authentication_error", "message": "invalid x-api-key"}})
        problem = invalid(body)
        if problem:
            status, kind, message = problem
            return self.reply(status, {"type": "error", "error": {"type": kind, "message": message}})
        result = respond(body)
        result.update({"id": "msg_fake", "type": "message", "role": "assistant", "model": body.get("model")})
        self.reply(200, result)

    def reply(self, status, obj):
        data = json.dumps(obj).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, *args):
        pass


if __name__ == "__main__":
    ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), Handler).serve_forever()
