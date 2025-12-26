#!/bin/bash
# Test the cz-bump workflow locally using act
# Usage: ./scripts/test-cz-bump.sh
# Example: ./scripts/test-cz-bump.sh

set -euo pipefail

mode="${1:-act}"

# EnsureDocker starts Docker Desktop on macOS and waits until Docker is ready.
EnsureDocker() {
	if docker info >/dev/null 2>&1; then
		return 0
	fi

	if [ "$(uname -s)" = "Darwin" ]; then
		open -a Docker >/dev/null 2>&1 || true
	fi

	i=0
	while [ $i -lt 45 ]; do
		if docker info >/dev/null 2>&1; then
			return 0
		fi
		sleep 2
		i=$((i + 1))
	done

	return 1
}

if [ "$mode" = "cli" ]; then
	echo "🧪 Testing Commitizen CLI bump (dry-run)"
	echo ""

	if ! EnsureDocker; then
		echo "❌ Docker is required for the containerized CLI dry-run."
		echo "Start Docker Desktop, then re-run: ./scripts/test-cz-bump.sh cli"
		exit 1
	fi

	docker run --rm \
		-v "$PWD:/app" \
		-w /app \
		python:3.12-slim \
		bash -lc "export DEBIAN_FRONTEND=noninteractive && apt-get update -qq && apt-get install -y -qq --no-install-recommends git && pip install -q commitizen==4.10.1 cz-conventional-gitmoji && cz bump --yes --dry-run --changelog --changelog-to-stdout > body.md"

	echo ""
	echo "✅ CLI dry-run completed!"
	echo "body.md written in repo root."
	exit 0
fi

echo "🧪 Testing bumpversion workflow (dry-run)"
echo ""

# Ensure docker is reachable for act.
if ! EnsureDocker; then
	echo "❌ Docker is required for act."
	echo "Start Docker Desktop, then re-run: ./scripts/test-cz-bump.sh"
	exit 1
fi

# Run act with push event
secretsArgs=()
if [ -n "${PERSONAL_ACCESS_TOKEN:-}" ]; then
	secretsArgs+=("-s" "PERSONAL_ACCESS_TOKEN=${PERSONAL_ACCESS_TOKEN}")
fi
if [ -n "${GITHUB_TOKEN:-}" ]; then
	secretsArgs+=("-s" "GITHUB_TOKEN=${GITHUB_TOKEN}")
fi

if [ "${#secretsArgs[@]}" -gt 0 ]; then
	act -n push \
		-W .github/workflows/bumpversion.yml \
		"${secretsArgs[@]}" \
		--no-cache-server \
		--container-architecture linux/amd64 \
		--verbose
else
	act -n push \
		-W .github/workflows/bumpversion.yml \
		--no-cache-server \
		--container-architecture linux/amd64 \
		--verbose
fi

echo ""
echo "✅ Test completed!"
echo "Note: Authentication/push/release steps may fail locally (expected behavior)"
