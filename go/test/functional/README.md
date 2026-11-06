# Functional tests

`run.py` drives a **running** MBFlow stack through its REST API: it creates the workflows in `pipelines/*.json`, runs each case and checks statuses, node results and outputs. It needs only Python 3.

| Pipeline | What it covers |
|---|---|
| `01_http_jq` | HTTP call to a third-party API, jq with a template from the input |
| `02_branching_merge` | Conditional true/false branches on a 1.4 MB dataset, skipped nodes, merge |
| `03_parallel_csv_join` | Two CSV downloads in parallel, CSV parsing, multi-parent jq join, report |
| `04_subworkflow_foreach` | Sub-workflow per item (`for_each`) with parallelism |
| `05_llm_summary` | LLM summary of a fetched document (Anthropic) |
| `06_llm_structured_routing` | LLM structured output (JSON schema) routed through a conditional to a second LLM call |
| `07_llm_agent_tools` | LLM agent that calls the built-in `http_request` tool in an auto tool-calling loop |
| `08_llm_map_reduce` | LLM inside a child workflow run per item (`for_each`, in parallel), answers scored against the dataset with jq, then an LLM summary |
| `09_llm_review_loop` | Writer, jq check and revision request in a loop edge: the check needs a code word that only the rejection feedback reveals, so a pass proves the loop fired once and the model used the feedback |
| `10_llm_agent_subworkflow_tool` | LLM agent whose tool is another workflow (`sub_workflow` function with `input_mapping` and `output_extractor`) |
| `11_llm_ticket_routing` | LLM classification (structured output) routed by edge conditions to three handlers, one of them a second LLM call; three cases |
| `12_llm_parallel_compare` | Two documents summarized by parallel LLM nodes, compared by an LLM node with two parents |
| `13_llm_failures` | An unknown model fails the execution with the provider's error; the next node does not run |

The data sources are public datasets on `raw.githubusercontent.com`.

## Running

```bash
cd go
MBFLOW_URL=http://localhost:8585 MBFLOW_EMAIL=admin@example.com MBFLOW_PASSWORD='...' \
ANTHROPIC_API_KEY=sk-ant-... python3 test/functional/run.py
```

- Pass names to run a subset: `run.py 02 06`. `VERBOSE=1` prints execution outputs.
- A case's `expect` can check the execution `status`, node statuses (`nodes`, and `not_nodes` for a status a node must not have), text in the execution `error` (`error_contains`), counts of execution events from `/executions/{id}/logs` (`events`, e.g. `{"loop.iteration": 1}`) and values in the execution or node output (`output`, with `eq`, `contains`, `icontains`, `regex`, `gt`, `gte`, `len`, `len_gte`, `type`, `nonempty`, `in`, `missing`).
- `MBFLOW_LLM_MODEL` selects the Claude model (default `claude-haiku-5-5`). The suite passes on `claude-haiku-5-5`, `claude-sonnet-5-5` and `claude-opus-5-5`.
- Without `ANTHROPIC_API_KEY` the LLM pipelines are skipped. With it, the key is stored as an encrypted credential, attached to the LLM workflows as the alias `anthropic` and used as `{{resource.anthropic.api_key}}`. The runner fails if the key appears in any API response.

## Dry run without a key

`fake_anthropic.py` stands in for the Messages API with canned answers. It checks MBFlow's LLM path (request format, credentials, tool loop, structured output), not model quality. Pipelines marked `"dry_run": false` check what a real model knows and are skipped in a dry run. Like the real API it rejects sampling parameters for current models and forced `tool_choice` for the models that refuse it (set `MBFLOW_LLM_MODEL` to try one), and it returns a thinking block before tool calls. The server must be allowed to call localhost (`MBFLOW_HTTP_ALLOWED_HOSTS=127.0.0.1`).

```bash
python3 test/functional/fake_anthropic.py 18080 dry-run-key &
ANTHROPIC_API_KEY=dry-run-key MBFLOW_LLM_BASE_URL=http://127.0.0.1:18080 MBFLOW_PASSWORD='...' \
  python3 test/functional/run.py
```
