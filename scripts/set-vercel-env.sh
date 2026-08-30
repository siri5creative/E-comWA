#!/usr/bin/env bash
# Run this from the repo root: bash scripts/set-vercel-env.sh
# Pushes local api/.env and web/.env.local values to Vercel (production +
# preview). Only pushes vars that have a non-empty local value — optional
# ones left blank locally (Firebase, payment gateway key, POS key, WA
# number) are skipped, matching the app's own graceful-degradation design.
set -euo pipefail

SCOPE="siri5creative-5999s-projects"
PROD_URL="https://e-comwa.vercel.app"

set -a
source api/.env
source web/.env.local
set +a

# GO_API_URL isn't in api/.env or web/.env.local's local values (those
# point at localhost) — production needs the deployed domain instead, so
# it's set explicitly here rather than sourced.
GO_API_URL="$PROD_URL"

push() {
  local var="$1"
  local val="${!var:-}"
  if [ -z "$val" ]; then
    echo "skip $var (empty locally)"
    return
  fi
  for env in production preview; do
    printf '%s' "$val" | vercel env add "$var" "$env" --force --scope "$SCOPE" >/dev/null
    echo "set $var [$env]"
  done
}

for var in DATABASE_URL SUPABASE_URL SUPABASE_JWT_SECRET SUPABASE_SERVICE_ROLE_KEY \
           NEXT_PUBLIC_SUPABASE_URL NEXT_PUBLIC_SUPABASE_ANON_KEY GO_API_URL; do
  push "$var"
done

echo "Done. Redeploy to pick up the new values: vercel deploy --prod --scope $SCOPE"
