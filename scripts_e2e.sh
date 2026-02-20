#!/usr/bin/env bash
set -euo pipefail

resp=$(curl -sS -X POST http://localhost:8080/api/chat/send \
  -H 'Content-Type: application/json' \
  -d '{"session_id":"demo","prompt":"你好"}')
req_id=$(echo "$resp" | sed -n 's/.*"request_id":"\([^"]*\)".*/\1/p')
[ -n "$req_id" ]

out=$(curl -sN "http://localhost:8080/api/chat/stream?request_id=${req_id}" | sed -n 's/^data: //p' | tr -d '\n')
echo "$out" | rg "百炼回复"
