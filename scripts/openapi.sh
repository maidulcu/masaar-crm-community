#!/usr/bin/env bash
# Regenerates the API docs from the handler annotations:
#   docs/docs.go, docs/swagger.json, docs/swagger.yaml  (Swagger 2.0, used by /docs)
#   docs/openapi.json                                   (OpenAPI 3.1, served at /openapi.json)
# Usage: scripts/openapi.sh [-check]   (-check fails if docs/openapi.json is stale)
set -euo pipefail
cd "$(dirname "$0")/.."

go run github.com/swaggo/swag/cmd/swag@v1.16.6 init \
  -d ./internal/api,./internal/api/handler -g doc.go -o docs \
  --parseDependency --parseInternal --quiet

go run ./cmd/openapi -in docs/swagger.json -out docs/openapi.json "$@"
