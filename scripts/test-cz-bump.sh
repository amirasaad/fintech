#!/bin/bash
# Test the cz-bump workflow locally using act
# Usage: ./scripts/test-cz-bump.sh
# Example: ./scripts/test-cz-bump.sh

echo "🧪 Testing bumpversion workflow (dry-run)"
echo ""

# Run act with push event
act -n push \
  -W .github/workflows/bumpversion.yml \
  -s PERSONAL_ACCESS_TOKEN=fake \
  -s GITHUB_TOKEN=fake \
  --container-architecture linux/amd64 \
  --verbose

echo ""
echo "✅ Test completed!"
echo "Note: Authentication/push/release steps may fail locally (expected behavior)"
