#!/usr/bin/env bash
# One-time: create the Terraform state bucket for dev. Safe to re-run.
# Usage: infra/scripts/bootstrap-state.sh [project-id] [location]
set -euo pipefail

PROJECT_ID="${1:-voyagami-dev-512208}"
LOCATION="${2:-asia-south1}"
BUCKET="${PROJECT_ID}-tfstate"

if gcloud storage buckets describe "gs://${BUCKET}" --project "${PROJECT_ID}" >/dev/null 2>&1; then
  echo "gs://${BUCKET} already exists"
else
  gcloud storage buckets create "gs://${BUCKET}" \
    --project "${PROJECT_ID}" \
    --location "${LOCATION}" \
    --uniform-bucket-level-access \
    --public-access-prevention
fi

gcloud storage buckets update "gs://${BUCKET}" --project "${PROJECT_ID}" --versioning
echo "state bucket ready: gs://${BUCKET}"
