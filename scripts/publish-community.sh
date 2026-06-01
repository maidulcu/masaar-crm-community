#!/usr/bin/env bash
# publish-community.sh — Export a clean community edition to a separate public repo.
# Run from the root of the private (full) repo.
#
# Usage:
#   ./scripts/publish-community.sh <path-to-public-repo-clone>
#
# Example:
#   git clone git@github.com:maidulcu/masaar-crm-community.git ../masaar-crm-community
#   ./scripts/publish-community.sh ../masaar-crm-community

set -euo pipefail

PRIVATE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PUBLIC="${1:-}"

if [[ -z "$PUBLIC" ]]; then
  echo "Usage: $0 <path-to-community-repo>"
  exit 1
fi

if [[ ! -d "$PUBLIC/.git" ]]; then
  echo "Error: $PUBLIC is not a git repo. Clone the public repo first."
  exit 1
fi

echo "▶ Syncing files from private repo to community repo..."

# Copy everything except private-repo-only paths
rsync -av --delete \
  --exclude='.git/' \
  --exclude='scripts/community-stubs/' \
  --exclude='internal/bos24/' \
  --exclude='internal/billing/stripe.go' \
  --exclude='.env' \
  --exclude='.env.*' \
  --exclude='*.pem' \
  --exclude='*.key' \
  --exclude='docker-compose.server.yml' \
  --exclude='deploy.sh' \
  --exclude='deploy-api.sh' \
  --exclude='deploy-web.sh' \
  --exclude='setup-git-automation.sh' \
  --exclude='server/' \
  --exclude='backups/' \
  --exclude='OPERATIONS.md' \
  --exclude='SAAS-PLAN.md' \
  --exclude='GTM-PLAN.md' \
  --exclude='DEPLOYMENT.md' \
  --exclude='DEPLOYMENT_README.md' \
  --exclude='DEPLOYMENT_LOCAL.md' \
  --exclude='COMPLETE_SETUP_GUIDE.md' \
  --exclude='GIT_AUTOMATION.md' \
  --exclude='MVP_OPTIMIZATION.md' \
  --exclude='SHARED_SERVICES.md' \
  --exclude='CLAUDE.md' \
  --exclude='CLAUDE_DESIGN.md' \
  --exclude='SYSTEM.md' \
  --exclude='k8s/' \
  "$PRIVATE/" "$PUBLIC/"

echo "▶ Replacing pro implementations with community stubs..."

# AI package — replace with no-op stubs, remove Ollama/Gemini
cp "$PRIVATE/scripts/community-stubs/ai/client.go"                        "$PUBLIC/internal/ai/client.go"
cp "$PRIVATE/scripts/community-stubs/ai/scoring.go"                       "$PUBLIC/internal/ai/scoring.go"
cp "$PRIVATE/scripts/community-stubs/ai/tagging.go"                       "$PUBLIC/internal/ai/tagging.go"
cp "$PRIVATE/scripts/community-stubs/ai/payment_confirmation_service.go"  "$PUBLIC/internal/ai/payment_confirmation_service.go"
cp "$PRIVATE/scripts/community-stubs/ai/payment_reminder_service.go"      "$PUBLIC/internal/ai/payment_reminder_service.go"
rm -f "$PUBLIC/internal/ai/ollama.go"
rm -f "$PUBLIC/internal/ai/gemini.go"
rm -f "$PUBLIC/internal/ai/methods.go"

# PDF package — type stubs only, no gofpdf
cp "$PRIVATE/scripts/community-stubs/pdf/invoice.go"         "$PUBLIC/internal/pdf/invoice.go"
cp "$PRIVATE/scripts/community-stubs/pdf/brochure.go"        "$PUBLIC/internal/pdf/brochure.go"
cp "$PRIVATE/scripts/community-stubs/pdf/property_report.go" "$PUBLIC/internal/pdf/property_report.go"

# Billing — remove Stripe
cp "$PRIVATE/scripts/community-stubs/billing/stripe.go" "$PUBLIC/internal/billing/stripe.go"

# Handlers — AI and BOS24 return 402
cp "$PRIVATE/scripts/community-stubs/handler/ai.go"                "$PUBLIC/internal/api/handler/ai.go"
cp "$PRIVATE/scripts/community-stubs/handler/bos24_integration.go" "$PUBLIC/internal/api/handler/bos24_integration.go"

# Remove BOS24 package entirely (handler nil-checks handle the rest)
rm -rf "$PUBLIC/internal/bos24/"

echo "▶ Cleaning go.mod dependencies..."
cd "$PUBLIC"
go mod edit -droprequire github.com/jung-kurt/gofpdf          2>/dev/null || true
go mod edit -droprequire github.com/stripe/stripe-go/v76      2>/dev/null || true
go mod edit -droprequire github.com/google/generative-ai-go   2>/dev/null || true
go mod tidy

echo "▶ Verifying community build compiles..."
go build ./...
echo "✓ Build passed"

echo ""
echo "▶ Files ready in: $PUBLIC"
echo "  Review the diff, then commit and push:"
echo "    cd $PUBLIC"
echo "    git add -A"
echo "    git commit -m 'chore: release community edition $(date +%Y-%m-%d)'"
echo "    git push"
