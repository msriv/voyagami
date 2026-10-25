#!/usr/bin/env bash
# Gives the dev `auth-secret` a random value if it has none. Run it once, after
# the secret exists and before Cloud Run is pointed at it. It never replaces an
# existing value; to rotate, see infra/README.md.
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-voyagami-dev-512208}"
SECRET="${SECRET:-auth-secret}"

if gcloud secrets versions list "$SECRET" --project "$PROJECT_ID" --filter 'state=enabled' --limit 1 --format 'value(name)' | grep -q .; then
  echo "${SECRET} already has a value, leaving it alone"
  exit 0
fi
openssl rand -hex 32 | tr -d '\n' | gcloud secrets versions add "$SECRET" --project "$PROJECT_ID" --data-file=-
echo "Added a random value to ${SECRET}"
