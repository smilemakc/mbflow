#!/usr/bin/env python3
"""Functional test suite for a running MBFlow stack.

Creates the workflows described in pipelines/*.json through the REST API,
runs every case and checks the results. Uses only the Python standard
library.

Usage:
    MBFLOW_URL=http://localhost:8585 \\
    MBFLOW_EMAIL=admin@example.com MBFLOW_PASSWORD=... \\
    ANTHROPIC_API_KEY=... \\
    python3 test/functional/run.py [name-filter ...]

Set MBFLOW_LLM_BASE_URL to send llm nodes to a stand-in API instead
(dry run, see fake_anthropic.py).

Pipelines that list "anthropic" in "requires" are skipped when
ANTHROPIC_API_KEY is not set, and pipelines with "dry_run": false (their
checks depend on what a real model knows) are skipped in dry runs. The key is stored as an encrypted credential
and attached to those workflows as the resource alias "anthropic", so
nodes use it as {{resource.anthropic.api_key}}; it never appears in a
workflow definition or execution input.
"""
import glob
import json
import os
import re
import sys
import time
import urllib.error
import urllib.request

BASE = os.environ.get("MBFLOW_URL", "http://localhost:8585").rstrip("/") + "/api/v1"
EMAIL = os.environ.get("MBFLOW_EMAIL", "admin@example.com")
PASSWORD = os.environ.get("MBFLOW_PASSWORD", "")
LLM_MODEL = os.environ.get("MBFLOW_LLM_MODEL", "claude-haiku-5-5")
RUN_TIMEOUT = int(os.environ.get("MBFLOW_RUN_TIMEOUT", "300"))
# Dry runs point llm nodes at a stand-in API (see fake_anthropic.py).
LLM_BASE_URL = os.environ.get("MBFLOW_LLM_BASE_URL", "")
HERE = os.path.dirname(os.path.abspath(__file__))

TOKEN = None
SECRETS = []  # values that must never show up in API responses


class Failure(Exception):
    pass


def api(method, path, body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(BASE + path, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if TOKEN:
        req.add_header("Authorization", "Bearer " + TOKEN)
    try:
        with urllib.request.urlopen(req, timeout=RUN_TIMEOUT) as resp:
            text = resp.read().decode()
            status = resp.status
    except urllib.error.HTTPError as err:
        text = err.read().decode()
        status = err.code
    for secret in SECRETS:
        if secret and secret in text:
            raise Failure(f"secret value leaked in response of {method} {path}")
    try:
        return status, json.loads(text) if text else None
    except ValueError:
        return status, text


def login():
    global TOKEN
    status, body = api("POST", "/auth/login", {"email": EMAIL, "password": PASSWORD})
    if status != 200:
        sys.exit(f"login failed: {status} {body}")
    TOKEN = body["access_token"]


def substitute(value, refs):
    """Replaces ${child:key} and ${llm_model} placeholders in strings."""
    if isinstance(value, dict):
        return {k: substitute(v, refs) for k, v in value.items()}
    if isinstance(value, list):
        return [substitute(v, refs) for v in value]
    if isinstance(value, str):
        value = value.replace("${llm_model}", LLM_MODEL)
        for key, wf_id in refs.items():
            value = value.replace("${child:" + key + "}", wf_id)
    return value


def create_workflow(definition, refs, suffix):
    wf = substitute(dict(definition), refs)
    wf["name"] = f"{wf['name']} [{suffix}]"
    if LLM_BASE_URL:
        for node in wf.get("nodes", []):
            if node.get("type") == "llm":
                node["config"]["base_url"] = LLM_BASE_URL
    status, body = api("POST", "/workflows", wf)
    if status not in (200, 201):
        raise Failure(f"create workflow {wf['name']!r}: {status} {body}")
    return body["id"]


def attach(wf_id, resource_id, alias):
    status, body = api("POST", f"/workflows/{wf_id}/resources", {"resource_id": resource_id, "alias": alias})
    if status not in (200, 201):
        raise Failure(f"attach {alias}: {status} {body}")


def run_case(wf_id, case):
    payload = {"input": case.get("input", {})}
    status, body = api("POST", f"/workflows/{wf_id}/execute", payload)
    if status not in (200, 202):
        raise Failure(f"execute: {status} {body}")
    ex_id = body["id"]
    deadline = time.time() + RUN_TIMEOUT
    while time.time() < deadline:
        status, ex = api("GET", f"/executions/{ex_id}")
        if ex.get("status") in ("completed", "failed", "cancelled", "timeout"):
            return ex
        time.sleep(1)
    raise Failure(f"execution {ex_id} did not finish in {RUN_TIMEOUT}s")


def get_path(value, path):
    for part in path.split(".") if path else []:
        if isinstance(value, list):
            value = value[int(part)]
        elif isinstance(value, dict):
            if part not in value:
                raise KeyError(path)
            value = value[part]
        else:
            raise KeyError(path)
    return value


def check(ex, expect):
    problems = []
    want_status = expect.get("status", "completed")
    if ex["status"] != want_status:
        problems.append(f"status {ex['status']!r} != {want_status!r} (error: {ex.get('error')!r})")
    nodes = {ne.get("node_id"): ne for ne in ex.get("node_executions") or []}
    for node_id, want in (expect.get("nodes") or {}).items():
        got = nodes.get(node_id, {}).get("status")
        if got != want:
            err = nodes.get(node_id, {}).get("error")
            problems.append(f"node {node_id}: status {got!r} != {want!r} {('(' + err + ')') if err else ''}")
    for node_id, unwanted in (expect.get("not_nodes") or {}).items():
        if nodes.get(node_id, {}).get("status") == unwanted:
            problems.append(f"node {node_id}: status must not be {unwanted!r}")
    if expect.get("events"):
        _, body = api("GET", f"/executions/{ex['id']}/logs")
        counts = {}
        for entry in (body or {}).get("logs") or []:
            counts[entry.get("event_type")] = counts.get(entry.get("event_type"), 0) + 1
        for event_type, want in expect["events"].items():
            if counts.get(event_type, 0) != want:
                problems.append(f"{counts.get(event_type, 0)} {event_type} events != {want}")
    if "error_contains" in expect and expect["error_contains"] not in (ex.get("error") or ""):
        problems.append(f"error {ex.get('error')!r} does not contain {expect['error_contains']!r}")
    for rule in expect.get("output") or []:
        source = ex.get("output")
        if "node" in rule:
            source = nodes.get(rule["node"], {}).get("output")
        op, want = rule.get("op", "eq"), rule.get("value")
        try:
            got = get_path(source, rule["path"])
        except (KeyError, IndexError, ValueError):
            if op != "missing":
                problems.append(f"{rule.get('node', 'output')}.{rule['path']}: missing")
            continue
        if op == "missing":
            problems.append(f"{rule.get('node', 'output')}.{rule['path']} should be absent: got {json.dumps(got, ensure_ascii=False)[:200]}")
            continue
        ok = {
            "eq": lambda: got == want,
            "contains": lambda: isinstance(got, (str, list)) and want in got,
            "icontains": lambda: isinstance(got, str) and str(want).lower() in got.lower(),
            "gt": lambda: isinstance(got, (int, float)) and got > want,
            "gte": lambda: isinstance(got, (int, float)) and got >= want,
            "len": lambda: hasattr(got, "__len__") and len(got) == want,
            "len_gte": lambda: hasattr(got, "__len__") and len(got) >= want,
            "type": lambda: type(got).__name__ == want,
            "nonempty": lambda: bool(got),
            "in": lambda: got in want,
            "regex": lambda: isinstance(got, str) and re.search(want, got) is not None,
        }[op]()
        if not ok:
            problems.append(f"{rule.get('node', 'output')}.{rule['path']} {op} {want!r}: got {json.dumps(got, ensure_ascii=False)[:200]}")
    return problems


def main(filters):
    if not PASSWORD:
        sys.exit("set MBFLOW_PASSWORD")
    login()
    anthropic_key = os.environ.get("ANTHROPIC_API_KEY", "")
    credential_id = None
    if anthropic_key:
        SECRETS.append(anthropic_key)
        status, body = api("POST", "/credentials/api-key", {
            "name": f"anthropic-functional-{int(time.time())}", "provider": "anthropic", "api_key": anthropic_key,
        })
        if status not in (200, 201):
            sys.exit(f"create credential: {status} {body}")
        credential_id = body.get("id") or body.get("data", {}).get("id")

    suffix = time.strftime("%Y%m%d-%H%M%S")
    results = []
    for path in sorted(glob.glob(os.path.join(HERE, "pipelines", "*.json"))):
        spec = json.load(open(path))
        name = os.path.basename(path)[:-5]
        if filters and not any(f in name for f in filters):
            continue
        if "anthropic" in spec.get("requires", []) and not credential_id:
            results.append((name, "SKIP", ["ANTHROPIC_API_KEY not set"], 0))
            continue
        if LLM_BASE_URL and spec.get("dry_run") is False:
            results.append((name, "SKIP", ["needs a real model (dry run)"], 0))
            continue
        started = time.time()
        problems = []
        try:
            refs = {}
            for key, child in (spec.get("children") or {}).items():
                refs[key] = create_workflow(child, refs, suffix)
                if credential_id and "anthropic" in spec.get("requires", []):
                    attach(refs[key], credential_id, "anthropic")
            wf_id = create_workflow(spec["workflow"], refs, suffix)
            if credential_id and "anthropic" in spec.get("requires", []):
                attach(wf_id, credential_id, "anthropic")
            for case in spec["cases"]:
                ex = run_case(wf_id, case)
                for p in check(ex, case.get("expect", {})):
                    problems.append(f"[{case['name']}] {p}")
                if os.environ.get("VERBOSE"):
                    print(json.dumps(ex.get("output"), ensure_ascii=False, indent=2)[:3000])
        except Failure as err:
            problems.append(str(err))
        results.append((name, "FAIL" if problems else "PASS", problems, time.time() - started))
        print(f"{results[-1][1]:4}  {name}  ({results[-1][3]:.1f}s)")
        for p in problems:
            print(f"      {p}")

    print()
    for name, verdict, problems, secs in results:
        if verdict == "SKIP":
            print(f"SKIP  {name}: {problems[0]}")
    failed = [r for r in results if r[1] == "FAIL"]
    if LLM_BASE_URL:
        print(f"\nDRY RUN: llm nodes used {LLM_BASE_URL}, not the real provider")
    print(f"\n{len(results)} pipelines: {sum(r[1] == 'PASS' for r in results)} passed, "
          f"{len(failed)} failed, {sum(r[1] == 'SKIP' for r in results)} skipped")
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main(sys.argv[1:])
