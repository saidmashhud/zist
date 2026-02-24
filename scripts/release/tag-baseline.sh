#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if [[ ! -d .git ]]; then
  echo "ERROR: not a git repository: $ROOT_DIR" >&2
  exit 1
fi

dirty_count="$(git status --porcelain | wc -l | tr -d ' ')"
if [[ "$dirty_count" != "0" ]]; then
  echo "ERROR: git worktree is dirty (${dirty_count} files). Clean/stage/commit first." >&2
  git status --short >&2
  exit 2
fi

TAG="${TAG:-zist/v$(date -u +%Y%m%d-%H%M%S)}"
if git rev-parse -q --verify "refs/tags/${TAG}" >/dev/null; then
  echo "ERROR: tag already exists: ${TAG}" >&2
  exit 1
fi

LATEST_GATE="$(ls -1t .artifacts/prod-gate/*.json 2>/dev/null | head -n1 || true)"
LATEST_DR="$(ls -1t .artifacts/dr-drill/*.json 2>/dev/null | head -n1 || true)"

if [[ -z "$LATEST_GATE" || -z "$LATEST_DR" ]]; then
  echo "ERROR: missing gate artifacts. Run prod gate and DR drill first." >&2
  exit 1
fi

BASELINE_DIR="$ROOT_DIR/.baseline"
mkdir -p "$BASELINE_DIR"
TS="$(date -u +%Y%m%d-%H%M%S)"
BASELINE_FILE="$BASELINE_DIR/zist-baseline-${TS}.md"

BRANCH="$(git rev-parse --abbrev-ref HEAD)"
COMMIT="$(git rev-parse HEAD)"
COMMIT_SHORT="$(git rev-parse --short HEAD)"
COMMIT_DATE="$(git show -s --format=%cI HEAD)"

cat > "$BASELINE_FILE" <<EOF
# Zist Release Baseline

- Generated (UTC): $(date -u +'%Y-%m-%d %H:%M:%S')
- Repo: \`$ROOT_DIR\`
- Branch: \`$BRANCH\`
- HEAD: \`$COMMIT_SHORT\` (\`$COMMIT\`)
- Commit date: \`$COMMIT_DATE\`
- Tag: \`$TAG\`

## Required Artifacts

- Prod gate: \`$LATEST_GATE\`
- DR drill: \`$LATEST_DR\`

## SLO Targets (Release Gate)

- service_availability >= 99.90%
- service_probe_p95 <= 1.0s
- restore RTO <= 900s
- restore RPO <= 300s

## Rollback/Migration Runbooks

- /Users/saidmashhud/Projects/personal/zist/docs/ops/incident-runbook.md
- /Users/saidmashhud/Projects/personal/zist/docs/ops/release-runbook.md
EOF

git tag -a "$TAG" -m "zist release baseline ${TS}"

echo "BASELINE_FILE=$BASELINE_FILE"
echo "TAG_CREATED=$TAG"
