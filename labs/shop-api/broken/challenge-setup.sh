#!/usr/bin/env bash
# Setup for challenge 6.C, "Troubleshoot a failing deploy".
#
# 1. Deploys the shop (challenge.yaml), sets its SIGNING_KEY secret and its
#    health check, and waits until it's Running. That's the good release.
# 2. Waits over a minute, so both releases show in deploy history.
# 3. Ships a new release with several faults in it. It doesn't say which.
#
# Needs csph, curl and jq, and:
#   COMPUTESPHERE_API_TOKEN   an API token (csph uses it too)
#   PROJECT_ID, ENVIRONMENT_ID the lab's project and environment
# Optional: COMPUTESPHERE_API_URL (default https://api.computesphere.com),
#   COMPUTESPHERE_ACCOUNT_ID, SERVICE_NAME (default shop), GAP_SECONDS (75),
#   DRY_RUN=1 to print the API calls instead of making them.
set -euo pipefail

API="${COMPUTESPHERE_API_URL:-https://api.computesphere.com}/v2"
NAME="${SERVICE_NAME:-shop}"
GAP="${GAP_SECONDS:-75}"
DRY="${DRY_RUN:-0}"
HERE="$(cd "$(dirname "$0")" && pwd)"

die() { echo "challenge-setup: $*" >&2; exit 1; }
say() { echo "==> $*"; }

for bin in csph curl jq; do command -v "$bin" >/dev/null || die "$bin is required"; done
[[ -n "${COMPUTESPHERE_API_TOKEN:-}" ]] || die "set COMPUTESPHERE_API_TOKEN to an API token"
[[ -n "${PROJECT_ID:-}" && -n "${ENVIRONMENT_ID:-}" ]] || die "set PROJECT_ID and ENVIRONMENT_ID"

# api METHOD PATH [JSON]
api() {
  local method=$1 path=$2 body=${3:-}
  if [[ "$DRY" == 1 ]]; then
    echo "DRY_RUN: $method $API$path ${body}" >&2
    echo '{"id":"dry-run","status":"Running","items":[{"id":"dry-run","name":"'"$NAME"'"}]}'
    return
  fi
  local args=(-sS --fail-with-body -X "$method" "$API$path"
    -H "Authorization: Bearer $COMPUTESPHERE_API_TOKEN" -H "Content-Type: application/json")
  [[ -n "${COMPUTESPHERE_ACCOUNT_ID:-}" ]] && args+=(-H "x-account-id: $COMPUTESPHERE_ACCOUNT_ID")
  [[ -n "$body" ]] && args+=(--data "$body")
  curl "${args[@]}" || die "$method $path failed"
}

wait_running() {
  local id=$1 status=""
  for _ in $(seq 1 60); do
    status=$(api GET "/deployments/$id" | jq -r '.status // ""' | tr '[:upper:]' '[:lower:]')
    [[ "$status" == running ]] && return 0
    [[ "$DRY" == 1 ]] && return 0
    sleep 10
  done
  die "the good release didn't reach Running in 10 minutes (last status: ${status:-unknown})"
}

say "Deploying the good release of '$NAME'"
if [[ "$DRY" == 1 ]]; then
  echo "DRY_RUN: csph deploy --file $HERE/challenge.yaml --project $PROJECT_ID --environment $ENVIRONMENT_ID --no-wait" >&2
else
  csph deploy --file "$HERE/challenge.yaml" --project "$PROJECT_ID" --environment "$ENVIRONMENT_ID" --no-wait >/dev/null
fi

ID=$(api GET "/deployments?environment_id=$ENVIRONMENT_ID" | jq -r --arg n "$NAME" '.items[] | select(.name == $n) | .id' | head -n1)
[[ -n "$ID" ]] || die "couldn't find the '$NAME' service in environment $ENVIRONMENT_ID"

KEY=$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')
api PUT "/deployments/$ID/env-vars" "$(jq -nc --arg k "$KEY" '{secret_vars: {SIGNING_KEY: $k}, skip_redeploy: true}')" >/dev/null
api PATCH "/deployments/$ID" '{"health_check":{"enabled":true,"path":"/healthz","port":8080,"initial_delay_seconds":100,"period_seconds":5},"skip_redeploy":true}' >/dev/null
api POST "/deployments/$ID/deploy" '{}' >/dev/null
say "Waiting for the good release to be Running (it takes about two minutes to start)"
wait_running "$ID"
say "Good release is Running"

say "Waiting ${GAP}s before the next release"
[[ "$DRY" == 1 ]] || sleep "$GAP"

say "Shipping the new release"
api PATCH "/deployments/$ID" '{"port":3000,"health_check":{"enabled":true,"path":"/healthz","port":3000,"initial_delay_seconds":5,"period_seconds":5},"skip_redeploy":true}' >/dev/null
api PUT "/deployments/$ID/env-vars" '{"secret_vars":{},"skip_redeploy":true}' >/dev/null
api POST "/deployments/$ID/scale" '{"spherelets":2,"skip_redeploy":true}' >/dev/null
api POST "/deployments/$ID/deploy" '{}' >/dev/null

cat <<EOF

The new release of '$NAME' is out, and the shop has stopped working.
Your job: get it answering 200 on /products again, on one spherelet, with its
original settings, by fixing the release or rolling back. No steps given.
EOF
