#!/usr/bin/env bash
# Usage: ./test-tools.sh <ollama-model-name>
# Passes if Ollama returns a real tool_calls entry (not plain text that looks like one).
set -euo pipefail
model="${1:?usage: $0 <ollama-model-name>}"
resp="$(curl -s http://localhost:11434/api/chat -d "{
  \"model\": \"$model\",
  \"stream\": false,
  \"messages\": [{\"role\": \"user\", \"content\": \"List the files in the current directory using the tool.\"}],
  \"tools\": [{\"type\": \"function\", \"function\": {\"name\": \"bash\", \"description\": \"Run a shell command\", \"parameters\": {\"type\": \"object\", \"properties\": {\"command\": {\"type\": \"string\"}}, \"required\": [\"command\"]}}}]
}")"
if echo "$resp" | grep -q '"tool_calls"'; then
  echo "PASS: $model returned a real tool call"
else
  echo "FAIL: no tool_calls in the response from $model"
  echo "$resp" | head -c 600; echo
  exit 1
fi
echo "speed: $(echo "$resp" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(round(d["eval_count"]/(d["eval_duration"]/1e9),1), "tokens/sec")' 2>/dev/null || echo unknown)"