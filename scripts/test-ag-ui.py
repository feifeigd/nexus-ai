#!/usr/bin/env python3
"""Verify the NexusAI Mock AG-UI stream directly or through CopilotKit Runtime."""

import argparse
import json
import sys
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

URLS = {
    "api": "http://127.0.0.1:8080/ag-ui",
    "runtime": "http://127.0.0.1:3000/api/copilotkit/agent/default/run",
}
EXPECTED_TYPES = [
    "RUN_STARTED",
    "TEXT_MESSAGE_START",
    "TEXT_MESSAGE_CONTENT",
    "TEXT_MESSAGE_CONTENT",
    "TEXT_MESSAGE_END",
    "RUN_FINISHED",
]
MOCK_PREFIX = "NexusAI Mock Agent 已收到："


def run_test(target: str, message: str) -> None:
    url = URLS[target]
    payload = {
        "threadId": "thread-python-test",
        "runId": "run-python-test",
        "messages": [
            {
                "id": "message-python-test",
                "role": "user",
                "content": message,
            }
        ],
    }
    request = Request(
        url,
        data=json.dumps(payload, ensure_ascii=False).encode("utf-8"),
        headers={
            "Content-Type": "application/json; charset=utf-8",
            "Accept": "text/event-stream",
        },
        method="POST",
    )

    print(f"POST {url}")
    print(f"Message: {message}")

    try:
        response = urlopen(request, timeout=30)
    except HTTPError as error:
        body = error.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"HTTP {error.code}: {body}") from error
    except (URLError, TimeoutError) as error:
        raise RuntimeError(
            f"Request failed; check that the {target} service is running: {error}"
        ) from error

    events = []
    with response:
        if response.status != 200:
            raise RuntimeError(f"Expected HTTP 200, got {response.status}")
        content_type = response.headers.get("Content-Type", "")
        if not content_type.startswith("text/event-stream"):
            raise RuntimeError(f"Expected an SSE response, got Content-Type: {content_type}")

        for raw_line in response:
            line = raw_line.decode("utf-8").strip()
            if line.startswith("data:"):
                events.append(json.loads(line[5:].strip()))

    actual_types = [event.get("type") for event in events]
    if actual_types != EXPECTED_TYPES:
        raise RuntimeError(f"Unexpected event sequence: {actual_types}")

    text = "".join(
        event.get("delta", "")
        for event in events
        if event.get("type") == "TEXT_MESSAGE_CONTENT"
    )
    expected_text = MOCK_PREFIX + message
    if text != expected_text:
        raise RuntimeError(f"Unexpected reply. Expected: {expected_text}; actual: {text}")

    for event in events:
        if event["type"] == "TEXT_MESSAGE_CONTENT":
            print(f"{event['type']}: {event.get('delta', '')}")
        else:
            print(event["type"])

    print("PASS: HTTP 200, SSE event sequence and reply text are correct.")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("target", choices=URLS, nargs="?", default="api")
    parser.add_argument("--message", default="你好，NexusAI")
    args = parser.parse_args()

    try:
        run_test(args.target, args.message)
    except (RuntimeError, ValueError, json.JSONDecodeError) as error:
        print(f"FAIL: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
