#!/usr/bin/env bash
# Setup for challenge 6.C, "Troubleshoot a failing deploy".
#
# 1. Deploys the shop (challenge.yaml), sets its SIGNING_KEY secret and its
#    health check, and waits until it's Running. That's the good release.
# 2. Lets it serve for a little over a minute.
# 3. Ships a new release with several faults in it. It doesn't say which.
#    Each one shows in something the learner can see: the URL, the logs,
#    the service's settings, its spherelet count or `csph services secrets list`.
#
# It only touches the one service it creates, in the project and environment
# you name. Use an API token with Project Access to that project only.
#
# Needs csph, curl and jq, and:
#   COMPUTESPHERE_API_TOKEN    an API token (csph uses it too)
#   PROJECT_ID, ENVIRONMENT_ID the lab's project and environment
#   the account ID: COMPUTESPHERE_ACCOUNT_ID if set, otherwise the default
#     account csph saved when you signed in, otherwise your only account
#     (`csph accounts list`). The script stops and says so if it can't tell.
# Optional: COMPUTESPHERE_API_URL (default https://api.computesphere.com),
#   SERVICE_NAME (default shop), GAP_SECONDS (75),
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

UUID_RE='^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'
ACCOUNT_HELP="find it with 'csph accounts list', then: export COMPUTESPHERE_ACCOUNT_ID=<account-id> and run this again"

# The account your lab project is in. The API needs it on every call.
resolve_account() {
  if [[ -n "${COMPUTESPHERE_ACCOUNT_ID:-}" ]]; then
    ACCOUNT_ID=$COMPUTESPHERE_ACCOUNT_ID; ACCOUNT_FROM="COMPUTESPHERE_ACCOUNT_ID"; return
  fi
  # csph saves your default account under common.account when you sign in
  # (or run `csph accounts use`).
  local conf="${CSPH_CONFIG_FILE:-$HOME/.config/computesphere/configuration.yaml}"
  if [[ -f "$conf" ]]; then
    ACCOUNT_ID=$(awk '/^common:/ {c=1; next} c && /^[^[:space:]]/ {c=0}
      c && $1 == "account:" {print $2; exit}' "$conf" | tr -d "\"'")
    [[ -n "$ACCOUNT_ID" ]] && { ACCOUNT_FROM="your csph default account"; return; }
  fi
  # Last try: if you have exactly one account, it's that one. (Skipped in a dry run.)
  if [[ "$DRY" != 1 ]]; then
    ACCOUNT_ID=$(csph accounts list -o json 2>/dev/null | jq -r '
      (if type == "array" then . else (.items // .data // []) end)
      | if length == 1 then .[0].id else empty end' 2>/dev/null || true)
    [[ -n "$ACCOUNT_ID" ]] && { ACCOUNT_FROM="your only account"; return; }
  fi
  die "couldn't work out your account ID; $ACCOUNT_HELP"
}

ACCOUNT_ID="" ACCOUNT_FROM=""
resolve_account
[[ "$ACCOUNT_ID" =~ $UUID_RE ]] || die "'$ACCOUNT_ID' (from $ACCOUNT_FROM) doesn't look like an account ID; $ACCOUNT_HELP"
say "Using account $ACCOUNT_ID (from $ACCOUNT_FROM)"

# api METHOD PATH [JSON]
api() {
  local method=$1 path=$2 body=${3:-}
  if [[ "$DRY" == 1 ]]; then
    echo "DRY_RUN: $method $API$path ${body}" >&2
    echo '{"id":"dry-run","status":"Running","items":[{"id":"dry-run","name":"'"$NAME"'"}]}'
    return
  fi
  local args=(-sS --fail-with-body -X "$method" "$API$path"
    -H "Authorization: Bearer $COMPUTESPHERE_API_TOKEN" -H "Content-Type: application/json"
    -H "x-account-id: $ACCOUNT_ID")
  [[ -n "$body" ]] && args+=(--data "$body")
  curl "${args[@]}" || die "$method $path failed. If the error is about the account, your lab project may be in another account: $ACCOUNT_HELP"
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
  COMPUTESPHERE_ACCOUNT_ID="$ACCOUNT_ID" csph deploy --file "$HERE/challenge.yaml" --project "$PROJECT_ID" --environment "$ENVIRONMENT_ID" --no-wait >/dev/null
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

say "Letting it serve for ${GAP}s before the next release"
[[ "$DRY" == 1 ]] || sleep "$GAP"

say "Shipping the new release"
api PATCH "/deployments/$ID" '{"port":3000,"health_check":{"enabled":true,"path":"/healthz","port":3000,"initial_delay_seconds":5,"period_seconds":5},"skip_redeploy":true}' >/dev/null
api PUT "/deployments/$ID/env-vars" '{"secret_vars":{},"skip_redeploy":true}' >/dev/null
api POST "/deployments/$ID/scale" '{"spherelets":2,"skip_redeploy":true}' >/dev/null
api POST "/deployments/$ID/deploy" '{}' >/dev/null

cat <<EOF

The new release of '$NAME' is out, and the shop has stopped working.
Your job: get it answering 200 on /products again, on one spherelet, with its
original settings, by fixing the release. No steps given.
EOF
